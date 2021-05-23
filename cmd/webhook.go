package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

const (
	admissionWebhookAnnotationStatusKey = "volume-permissions-container-injector-webhook.malston.me/status"
	configMapKey = "volumepermissions.yaml"
	initContainerTemplate = `initContainers:
- command:
  - /bin/bash
  - -ec
  - |-
    chown -R replace-permission:replace-permission /replace-mountPath
  image: docker.io/bitnami/bitnami-shell:10
  imagePullPolicy: Always
  name: volume-permissions
  securityContext:
    runAsUser: 0
  volumeMounts:
  - mountPath: /replace-mountPath
    name: replace-mountName
`
)

type WebhookServer struct {
	server        *http.Server
	clientset *kubernetes.Clientset
}

type Parameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	initContainerCfgFile string // path to initcontainer injector configuration file
}

type Config struct {
	InitContainers []corev1.Container `yaml:"initContainers"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
}

func loadConfig(configFile string) (*Config, error) {

	data := []byte(configFile)

	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Check whether the target resource needs to be mutated
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[admissionWebhookAnnotationStatusKey]

	// determine whether to perform mutation based on annotation for the target resource
	required := true
	if strings.ToLower(status) == "injected" {
		required = false
	}

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
	return required
}

func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value []corev1.Container
	for _, add := range added {
		value = append(value, add)
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

func (svr *WebhookServer) createUpdateConfigMap(ctx context.Context, name, namespace, content string) error {
	glog.Infof("creating configmap with data: %s", content)
	fqn := fmt.Sprintf("%s-%s", namespace, name)
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{configMapKey: content},
	}

	if _, err := svr.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, fqn, metav1.GetOptions{}); err != nil {
		if _, err = svr.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metav1.CreateOptions{}); err != nil {
			glog.Error(err, "unable to create ConfigMap", "configmap", cm)
			return err
		}
		return nil
	}

	return nil
}

// create mutation patch for resources
func createPatch(pod *corev1.Pod, initContainerConfig *Config, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, addContainer(pod.Spec.InitContainers, initContainerConfig.InitContainers, "/spec/initContainers")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

func replaceInitContainerStrings(pod corev1.Pod) string {
	if len(pod.Spec.Containers) > 0 {
		var container string
		var permission int64
		var mountPath, mountName string
		if len(pod.Spec.Containers[0].VolumeMounts) > 0 {
			if pod.Spec.Containers[0].SecurityContext != nil && pod.Spec.Containers[0].SecurityContext.RunAsGroup != nil {
				permission = *pod.Spec.Containers[0].SecurityContext.RunAsGroup
				container = strings.Replace(initContainerTemplate, "replace-permission", strconv.FormatInt(permission, 10), -1)
			}
			mountPath = pod.Spec.Containers[0].VolumeMounts[0].MountPath
			if container == ""{
				container = strings.Replace(initContainerTemplate, "/replace-mountPath", mountPath, -1)
			} else {
				container = strings.Replace(container, "/replace-mountPath", mountPath, -1)
			}
			mountName = pod.Spec.Containers[0].VolumeMounts[0].Name
			container = strings.Replace(container, "replace-mountName", mountName, -1)
		}
		if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.FSGroup != nil {
			permission = *pod.Spec.SecurityContext.FSGroup
			if container == ""{
				container = strings.Replace(initContainerTemplate, "replace-permission", strconv.FormatInt(permission, 10), -1)
			} else {
				container = strings.Replace(container, "replace-permission", strconv.FormatInt(permission, 10), -1)
			}
		}
		if container == "" || strings.Contains(container, "replace-") {
			return ""
		}
		return container
	}

	return ""
}

// main mutation process
func (svr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
		glog.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	initContainer := replaceInitContainerStrings(pod)
	if initContainer == "" {
		glog.Infof("Skipping mutation for %s/%s due to pod not containing a securityContext or volumes", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	//err := svr.createUpdateConfigMap(context.TODO(), fmt.Sprintf("%s-configmap", pod.Name), pod.Namespace, initContainer)
	//if err != nil {
	//	return &v1beta1.AdmissionResponse{
	//		Result: &metav1.Status{
	//			Message: err.Error(),
	//		},
	//	}
	//}

	initContainerConfig, err := loadConfig(initContainer)
	if err != nil {
		glog.Errorf("Failed to load configuration: %v", err)
	}
	glog.Infof("initContainerConfig: %+v", *initContainerConfig)
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, initContainerConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (svr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = svr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write admissionreview reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

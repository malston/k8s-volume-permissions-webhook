package main

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var initContainers = `initContainers:
- command:
  - /bin/bash
  - -ec
  - |-
    chown -R 1001:1001 /bitnami/redis/data
  image: docker.io/bitnami/bitnami-shell:10
  imagePullPolicy: Always
  name: volume-permissions
  securityContext:
    runAsUser: 0
  volumeMounts:
  - mountPath: /bitnami/redis/data
    name: redis-data
`

var posCases = []struct {
	pod *corev1.Pod
	initContainers bool
}{
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase1"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "redis-data",
								MountPath: "/bitnami/redis/data",
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				}},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase2"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "redis-data",
								MountPath: "/bitnami/redis/data",
							},
						},
						SecurityContext: &corev1.SecurityContext{RunAsGroup: func(i int64) *int64 { return &i }(1001)},
					},
				},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase3"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
					},
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "redis-data",
								MountPath: "/bitnami/redis/data",
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				}},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase4"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
					},
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "secret-volume-registry-credentials",
								MountPath: "/var/build-secrets/registry-credentials",
							},
							{
								Name:      "redis-data",
								MountPath: "/bitnami/redis/data",
							},
							{
								Name:      "service-account-token-z7cpd",
								MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
								ReadOnly:  true,
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				},
				Volumes: []corev1.Volume{
					{
						Name: "secret-volume-registry-credentials",
					},
					{
						Name: "redis-data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "redis",
								ReadOnly:  false,
							},
						},
					},
				},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase5"},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
					},
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "secret-volume-registry-credentials", MountPath: "/var/build-secrets/registry-credentials",
							},
							{
								Name: "redis-data", MountPath: "/bitnami/redis/data",
							},
							{
								Name: "service-account-token-z7cpd", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount", ReadOnly: true,
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				},
				Volumes: []corev1.Volume{
					{
						Name: "secret-volume-registry-credentials",
					},
					{
						Name: "redis-data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "redis",
								ReadOnly:  false,
							},
						},
					},
				},
			},
		},
		initContainers: true,
	},
}

var negCases = []struct {
	pod *corev1.Pod
}{
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase1"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				}},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase2"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						SecurityContext: &corev1.SecurityContext{RunAsGroup: func(i int64) *int64 { return &i }(1001)},
					},
				},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase3"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "redis-data", MountPath: "/bitnami/redis/data",
							},
						},
					},
				},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase4"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "redis-data", MountPath: "/bitnami/redis/data",
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase5"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "redis-data", MountPath: "/bitnami/redis/data",
							},
						},
						SecurityContext: &corev1.SecurityContext{},
					},
				},
			},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase6"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						VolumeMounts: []corev1.VolumeMount{
							{
								Name: "default-token-hw45h", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
							},
						},
					},
				},
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup: func(i int64) *int64 { return &i }(1001),
				}},
		},
	},
	{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "negative-testcase7"},
			Spec:       corev1.PodSpec{},
		},
	},
}

func TestReplaceInitContainerStrings(t *testing.T) {
	t.Run("positive cases", func(t *testing.T) {
		want := initContainers
		for _, c := range posCases {
			if len(c.pod.Spec.Containers) > 0 {
				got := replaceInitContainerStrings(c.pod.Spec.SecurityContext, c.pod.Spec.Containers, c.pod.Spec.Volumes)
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("replaceInitContainerStrings(%s) got %s want %s", c.pod.Name, got, want)
				}
			}
			if len(c.pod.Spec.InitContainers) > 0 {
				got := replaceInitContainerStrings(c.pod.Spec.SecurityContext, c.pod.Spec.InitContainers, c.pod.Spec.Volumes)
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("replaceInitContainerStrings(%s) got %s want %s", c.pod.Name, got, want)
				}
			}
		}
	})

	t.Run("negative cases", func(t *testing.T) {
		for _, c := range negCases {
			want := ""
			got := replaceInitContainerStrings(c.pod.Spec.SecurityContext, c.pod.Spec.Containers, c.pod.Spec.Volumes)
			if diff := cmp.Diff("", got); diff != "" {
				t.Errorf("replaceInitContainerStrings(%s) got %s want %s", c.pod.Name, got, want)
			}
		}
	})
}

func TestAddContainer(t *testing.T) {
	t.Run("positive cases", func(t *testing.T) {
		var want []patchOperation
		want = append(want, patchOperation{
			Op:    "add",
			Path:  "/spec/initContainers",
		})
		for _, c := range posCases {
			initContainerConfig, err := loadConfig(initContainers)
			assert.NoError(t, err)
			want[0].Value = initContainerConfig.InitContainers
			if c.initContainers {
				want[0].Op = "replace"
				want[0].Value = append(initContainerConfig.InitContainers, c.pod.Spec.InitContainers...)
			}
			got := addContainer(c.pod.Spec.InitContainers, initContainerConfig.InitContainers, "/spec/initContainers")
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("addContainer(%s) got %s want %s", c.pod.Name, got, want)
			}
		}
	})
}

func TestLoadConfig(t *testing.T) {
	want := &Config{
		InitContainers: []corev1.Container{
			{
				Name:            "volume-permissions",
				Image:           "docker.io/bitnami/bitnami-shell:10",
				ImagePullPolicy: "Always",
				Command:         []string{"/bin/bash", "-ec", "chown -R 1001:1001 /bitnami/redis/data"},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "redis-data", MountPath: "/bitnami/redis/data"},
				},
				SecurityContext: &corev1.SecurityContext{RunAsUser: func(i int64) *int64 { return &i }(0)},
			},
		},
	}
	got, err := loadConfig(initContainers)
	assert.NoError(t, err)
	if ok := cmp.Equal(want, got); !ok {
		t.Errorf("loadConfig(%s) got %v want %v", initContainers, got, want)
	}
}

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

func TestReplaceInitContainerStrings(t *testing.T) {
	posCases := []struct {
		pod *corev1.Pod
	}{
		{
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "positive-testcase1"},
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
									Name: "redis-data", MountPath: "/bitnami/redis/data",
								},
							},
							SecurityContext: &corev1.SecurityContext{RunAsGroup: func(i int64) *int64 { return &i }(1001)},
						},
					},
				},
			},
		},
	}
	t.Run("positive cases", func(t *testing.T) {
		want := initContainers
		for _, c := range posCases {
			got := replaceInitContainerStrings(*c.pod)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("replaceInitContainerStrings(%s) got %s want %s", c.pod.Name, got, want)
			}
		}
	})
	negCases := []struct {
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
				Spec: corev1.PodSpec{},
			},
		},
	}
	t.Run("negative cases", func(t *testing.T) {
		for _, c := range negCases {
			want := ""
			got := replaceInitContainerStrings(*c.pod)
			if diff := cmp.Diff("", got); diff != "" {
				t.Errorf("replaceInitContainerStrings(%s) got %s want %s", c.pod.Name, got, want)
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

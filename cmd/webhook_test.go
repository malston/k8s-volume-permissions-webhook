package main

import (
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestReplaceInitContainerStrings(t *testing.T) {
	want := `
initContainers:
- command:
  - /bin/bash
  - -ec
  - |
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
	posCases := []struct {
		pod  *corev1.Pod
	}{
		{
			pod: &corev1.Pod{
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
		for _, c := range posCases {
			got := replaceInitContainerStrings(*c.pod)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("replaceInitContainerStrings(%+v) got %s want %s", &c, got, want)
			}
		}
	})
	negCases := []struct {
		pod  *corev1.Pod
	}{
		{
			pod: &corev1.Pod{
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
				Spec: corev1.PodSpec{},
			},
		},
	}
	t.Run("negative cases", func(t *testing.T) {
		for _, c := range negCases {
			got := replaceInitContainerStrings(*c.pod)
			if diff := cmp.Diff("", got); diff != "" {
				t.Errorf("replaceInitContainerStrings(%+v) got %s want %s", &c, got, "")
			}
		}
	})
}

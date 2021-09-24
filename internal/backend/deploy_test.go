package backend

import (
	"reflect"
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/joyrex2001/kubedock/internal/model/types"
)

func TestStartContainer(t *testing.T) {
	tests := []struct {
		kub   *instance
		in    *types.Container
		state DeployState
		err   bool
	}{
		{ // deployment not created
			kub: &instance{
				namespace: "default",
				cli:       fake.NewSimpleClientset(),
			},
			in:    &types.Container{ID: "rc752", ShortID: "tr808", Name: "f1spirit"},
			state: DeployFailed,
			err:   true,
		},
		{ // deployment already exists
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tb303",
						Namespace: "default",
					},
				}),
			},
			in:    &types.Container{ID: "rc752", ShortID: "tb303", Name: "f1spirit"},
			state: DeployFailed,
			err:   true,
		},
	}

	for i, tst := range tests {
		state, err := tst.kub.StartContainer(tst.in)
		if err != nil && !tst.err {
			t.Errorf("failed test %d - unexpected error %s", i, err)
		}
		if err == nil && tst.err {
			t.Errorf("failed test %d - expected error, but succeeded instead", i)
		}
		if state != tst.state {
			t.Errorf("failed test %d - expected state %d, but got state %d", i, tst.state, state)
		}
	}
}

func TestWaitReadyState(t *testing.T) {
	tests := []struct {
		in    *types.Container
		kub   *instance
		state DeployState
		out   bool
	}{
		{
			kub: &instance{
				namespace: "default",
				cli:       fake.NewSimpleClientset(),
			},
			in:    &types.Container{Name: "f1spirit"},
			state: DeployFailed,
			out:   true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tb303",
						Namespace: "default",
					},
				}),
			},
			in:    &types.Container{Name: "f1spirit", ShortID: "tb303"},
			state: DeployFailed,
			out:   true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "f1spirit",
						Namespace: "default",
						Labels:    map[string]string{"kubedock.containerid": "tr909"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				}, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tr909",
						Namespace: "default",
					},
				}),
			},
			in:    &types.Container{ID: "rc752", ShortID: "tr909", Name: "f1spirit"},
			state: DeployFailed,
			out:   true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "f1spirit",
						Namespace: "default",
						Labels:    map[string]string{"kubedock.containerid": "tr808"},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{RestartCount: 1},
						},
					},
				}, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tr808",
						Namespace: "default",
					},
				}),
			},
			in:    &types.Container{ID: "rc752", ShortID: "tr808", Name: "f1spirit"},
			state: DeployFailed,
			out:   true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "f1spirit",
						Namespace: "default",
						Labels:    map[string]string{"kubedock.containerid": "tr909"},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"}}},
						},
					},
				}, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tr909",
						Namespace: "default",
					},
				}),
			},
			in:    &types.Container{ID: "rc752", ShortID: "tr909", Name: "f1spirit"},
			state: DeployCompleted,
			out:   false,
		},
	}

	for i, tst := range tests {
		state, err := tst.kub.waitReadyState(tst.in, 1)
		if (err != nil && !tst.out) || (err == nil && tst.out) {
			t.Errorf("failed test %d - unexpected return value %s", i, err)
		}
		if state != tst.state {
			t.Errorf("failed test %d - expected state %d, but got %d", i, tst.state, state)
		}
	}
}

func TestWaitInitContainerRunning(t *testing.T) {
	tests := []struct {
		in   *types.Container
		name string
		kub  *instance
		out  bool
	}{
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "f1spirit",
						Namespace: "default",
						Labels:    map[string]string{"kubedock.containerid": "rc752"},
					},
					Status: corev1.PodStatus{
						InitContainerStatuses: []corev1.ContainerStatus{
							{Name: "setup", State: corev1.ContainerState{Running: nil}},
						},
					},
				}),
			},
			name: "setup",
			in:   &types.Container{ID: "rc752", ShortID: "tr808", Name: "f1spirit"},
			out:  true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tb303",
						Namespace: "default",
						Labels:    map[string]string{"kubedock.containerid": "tb303"},
					},
					Status: corev1.PodStatus{
						InitContainerStatuses: []corev1.ContainerStatus{
							{Name: "setup", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
						},
					},
				}),
			},
			name: "setup",
			in:   &types.Container{ID: "rc752", ShortID: "tb303", Name: "f1spirit"},
			out:  false,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tr606",
						Namespace: "default",
						Labels:    map[string]string{"kubedock": "tr606"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				}),
			},
			name: "setup",
			in:   &types.Container{ID: "rc752", ShortID: "tr606", Name: "f1spirit"},
			out:  true,
		},
		{
			kub: &instance{
				namespace: "default",
				cli: fake.NewSimpleClientset(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tr606",
						Namespace: "default",
						Labels:    map[string]string{"kubedock": "tr606"},
					},
					Status: corev1.PodStatus{
						InitContainerStatuses: []corev1.ContainerStatus{
							{Name: "setup"},
						},
					},
				}),
			},
			name: "main",
			in:   &types.Container{ID: "rc752", ShortID: "tr606", Name: "f1spirit"},
			out:  true,
		},
	}

	for i, tst := range tests {
		res := tst.kub.waitInitContainerRunning(tst.in, tst.name, 1)
		if (res != nil && !tst.out) || (res == nil && tst.out) {
			t.Errorf("failed test %d - unexpected return value %s", i, res)
		}
	}
}

func TestAddVolumes(t *testing.T) {
	tests := []struct {
		in    *types.Container
		count int
	}{
		{in: &types.Container{}, count: 0},
		{in: &types.Container{Binds: []string{".:/remote:rw"}}, count: 1},
		{in: &types.Container{Binds: []string{".:/remote:rw", "deploy_test.go:/tmp/gogo.go"}}, count: 2},
		{in: &types.Container{Binds: []string{".:/remote:rw", "xxx:/tmp/gogo.go"}}, count: 1},
	}

	for i, tst := range tests {
		podtm := &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}},
			},
		}
		kub := &instance{cli: fake.NewSimpleClientset()}
		kub.addVolumes(tst.in, podtm)
		count := len(podtm.Spec.Volumes)
		if count != tst.count {
			t.Errorf("failed test %d - expected %d initContainers, but got %d", i, tst.count, count)
		}
	}
}

func TestContainerPorts(t *testing.T) {
	tests := []struct {
		in    *types.Container
		count int
	}{
		{in: &types.Container{}, count: 0},
		{in: &types.Container{ExposedPorts: map[string]interface{}{"909/tcp": 0}}, count: 1},
	}

	for i, tst := range tests {
		kub := &instance{}
		count := len(kub.getContainerPorts(tst.in))
		if count != tst.count {
			t.Errorf("failed test %d - expected %d container ports, but got %d", i, tst.count, count)
		}
	}
}

func TestGetLabels(t *testing.T) {
	tests := []struct {
		in    *types.Container
		count int
	}{
		{in: &types.Container{}, count: 3},
		{in: &types.Container{Labels: map[string]string{"computer": "msx"}}, count: 3},
	}

	for i, tst := range tests {
		kub := &instance{}
		count := len(kub.getLabels(tst.in))
		if count != tst.count {
			t.Errorf("failed test %d - expected %d labels, but got %d", i, tst.count, count)
		}
	}
}

func TestGetServices(t *testing.T) {
	tests := []struct {
		in    *types.Container
		svcs  int
		ports int
	}{
		{in: &types.Container{}, svcs: 0, ports: 0},
		{in: &types.Container{ExposedPorts: map[string]interface{}{"100/tcp": 1}}, svcs: 0, ports: 0},
		{in: &types.Container{ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{100: 200}}, svcs: 0, ports: 0},
		{in: &types.Container{ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{200: 200}}, svcs: 0, ports: 0},
		{in: &types.Container{NetworkAliases: []string{"tb303"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}}, svcs: 1, ports: 1},
		{in: &types.Container{NetworkAliases: []string{"tb303"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{100: 200}}, svcs: 1, ports: 1},
		{in: &types.Container{NetworkAliases: []string{"tb303"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{200: 200}}, svcs: 1, ports: 2},
		{in: &types.Container{NetworkAliases: []string{"tb303"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{-300: 300}}, svcs: 1, ports: 2},
		{in: &types.Container{NetworkAliases: []string{"tb303", "tr909"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}}, svcs: 2, ports: 1},
		{in: &types.Container{NetworkAliases: []string{"tb303", "tr909"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{100: 200}}, svcs: 2, ports: 1},
		{in: &types.Container{NetworkAliases: []string{"tb303", "tr909"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}, HostPorts: map[int]int{200: 200}}, svcs: 2, ports: 2},
		{in: &types.Container{NetworkAliases: []string{"tb303_"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}}, svcs: 0, ports: 0},
		{in: &types.Container{NetworkAliases: []string{"303"}, ExposedPorts: map[string]interface{}{"100/tcp": 1}}, svcs: 0, ports: 0},
	}
	for i, tst := range tests {
		kub := &instance{}
		res := kub.getServices(tst.in)
		count := len(res)
		if count != tst.svcs {
			t.Errorf("failed test %d - expected %d services, but got %d", i, tst.svcs, count)
		}
		if count > 0 && tst.ports > 0 && len(res[0].Spec.Ports) != tst.ports {
			t.Errorf("failed test %d - expected %d ports, but got %d", i, tst.ports, len(res[0].Spec.Ports))
		}
	}

	kub := &instance{}
	res := kub.getServices(&types.Container{
		NetworkAliases: []string{"tb303"},
		ExposedPorts:   map[string]interface{}{"100/tcp": 1},
		ImagePorts:     map[string]interface{}{"400/tcp": 1},
		HostPorts:      map[int]int{200: 200, -300: 300},
	})
	sort.Slice(res[0].Spec.Ports, func(i, j int) bool {
		return res[0].Spec.Ports[i].Name < res[0].Spec.Ports[j].Name
	})
	exp := []corev1.ServicePort{
		{Name: "tcp-100-100", Protocol: "TCP", Port: 100, TargetPort: intstr.IntOrString{IntVal: 100}},
		{Name: "tcp-200-200", Protocol: "TCP", Port: 200, TargetPort: intstr.IntOrString{IntVal: 200}},
		{Name: "tcp-300-300", Protocol: "TCP", Port: 300, TargetPort: intstr.IntOrString{IntVal: 300}},
		{Name: "tcp-400-400", Protocol: "TCP", Port: 400, TargetPort: intstr.IntOrString{IntVal: 400}},
	}
	if !reflect.DeepEqual(res[0].Spec.Ports, exp) {
		t.Errorf("failed detail ports test - expected %#v, but got %#v", res[0].Spec.Ports, exp)
	}
}

func TestGetAnnotations(t *testing.T) {
	tests := []struct {
		in    *types.Container
		count int
	}{
		{in: &types.Container{}, count: 1},
		{in: &types.Container{Labels: map[string]string{"computer": "msx"}}, count: 2},
	}

	for i, tst := range tests {
		kub := &instance{}
		count := len(kub.getAnnotations(tst.in))
		if count != tst.count {
			t.Errorf("failed test %d - expected %d labels, but got %d", i, tst.count, count)
		}
	}
}

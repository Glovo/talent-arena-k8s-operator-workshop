package v1

import (
	"context"
	"encoding/hex"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GetHash returns a hash for MyOllama object based on the spec
func (m *MyOllama) GetHash() string {
	hash := hex.EncodeToString([]byte(m.Spec.Model))
	if len(hash) > 10 {
		hash = hash[:10]
	}
	return hash
}

// ChildReplicaSet returns a new expected child ReplicaSet object based on MyOllama object's spec
func (m *MyOllama) ChildReplicaSet(ctx context.Context, scheme *runtime.Scheme) (appsv1.ReplicaSet, error) {
	log := log.FromContext(ctx)
	versionHash := m.GetHash()
	objLabels := map[string]string{
		"ollama-ref":  m.Name,
		"ollama-hash": versionHash,
	}
	objMeta := metav1.ObjectMeta{
		Name:      fmt.Sprintf("myollama-%s-%s", m.Name, versionHash),
		Namespace: m.Namespace,
		Labels:    objLabels,
	}

	rs := appsv1.ReplicaSet{
		ObjectMeta: objMeta,
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: objLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objMeta,
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "ollama",
							Image: "ollama/ollama:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 11434,
								},
							},
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								fmt.Sprintf(`#!/bin/bash
# Start Ollama in the background.
/bin/ollama serve &
# Record Process ID.
pid=$!
# Pause for Ollama to start.
sleep 5
echo "Retrieve model..."
ollama pull %s
echo "Done!"
# Wait for Ollama process to finish.
wait $pid`,
									m.Spec.Model),
							},
						},
					},
				},
			},
		},
	}
	// set owner reference
	if err := controllerutil.SetControllerReference(m, &rs, scheme); err != nil {
		log.Error(err, "Failed to set owner reference")
		return appsv1.ReplicaSet{}, err
	}
	return rs, nil
}

// ChildService returns a new expected child Service object based on MyOllama object's spec
func (m *MyOllama) ChildService(ctx context.Context, scheme *runtime.Scheme) (corev1.Service, error) {
	log := log.FromContext(ctx)
	versionHash := m.GetHash()
	objLabels := map[string]string{
		"ollama-ref":  m.Name,
		"ollama-hash": versionHash,
	}
	objMeta := metav1.ObjectMeta{
		Name:      fmt.Sprintf("myollama-%s", m.Name),
		Namespace: m.Namespace,
		Labels:    objLabels,
	}

	svc := corev1.Service{
		ObjectMeta: objMeta,
		Spec: corev1.ServiceSpec{
			Selector: objLabels,
			Ports: []corev1.ServicePort{
				{
					Port: 11434,
				},
			},
		},
	}
	// set owner reference
	if err := controllerutil.SetControllerReference(m, &svc, scheme); err != nil {
		log.Error(err, "Failed to set owner reference")
		return corev1.Service{}, err
	}
	return svc, nil
}

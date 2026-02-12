package secrets

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Builder struct {
	secret *corev1.Secret
}

func NewBuilder() *Builder {
	return &Builder{
		secret: &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{},
			Type:       corev1.SecretTypeOpaque,
		},
	}
}

func (b *Builder) WithName(name string) *Builder {
	b.secret.Name = name
	return b
}

func (b *Builder) WithNamespace(namespace string) *Builder {
	b.secret.Namespace = namespace
	return b
}

func (b *Builder) WithType(secretType corev1.SecretType) *Builder {
	b.secret.Type = secretType
	return b
}

func (b *Builder) WithData(data map[string][]byte) *Builder {
	if b.secret.Data == nil {
		b.secret.Data = make(map[string][]byte)
	}
	for key, value := range data {
		b.secret.Data[key] = value
	}
	return b
}

func (b *Builder) WithDataEntry(key string, value []byte) *Builder {
	if b.secret.Data == nil {
		b.secret.Data = make(map[string][]byte)
	}
	b.secret.Data[key] = value
	return b
}

func (b *Builder) WithStringData(data map[string]string) *Builder {
	if b.secret.StringData == nil {
		b.secret.StringData = make(map[string]string)
	}
	for key, value := range data {
		b.secret.StringData[key] = value
	}
	return b
}

func (b *Builder) WithStringDataEntry(key, value string) *Builder {
	if b.secret.StringData == nil {
		b.secret.StringData = make(map[string]string)
	}
	b.secret.StringData[key] = value
	return b
}

func (b *Builder) WithLabels(labels map[string]string) *Builder {
	if b.secret.Labels == nil {
		b.secret.Labels = make(map[string]string)
	}
	for key, value := range labels {
		b.secret.Labels[key] = value
	}
	return b
}

func (b *Builder) WithLabel(key, value string) *Builder {
	if b.secret.Labels == nil {
		b.secret.Labels = make(map[string]string)
	}
	b.secret.Labels[key] = value
	return b
}

func (b *Builder) WithAnnotations(annotations map[string]string) *Builder {
	if b.secret.Annotations == nil {
		b.secret.Annotations = make(map[string]string)
	}
	for key, value := range annotations {
		b.secret.Annotations[key] = value
	}
	return b
}

func (b *Builder) WithAnnotation(key, value string) *Builder {
	if b.secret.Annotations == nil {
		b.secret.Annotations = make(map[string]string)
	}
	b.secret.Annotations[key] = value
	return b
}

func (b *Builder) Build() *corev1.Secret {
	return b.secret
}

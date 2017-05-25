package elasticsearch

import (
	"bytes"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
)

func TestConfigGeneration(t *testing.T) {
	cfg, err := generateTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		testcfg, err := generateTestConfig()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(cfg, testcfg) {
			t.Fatalf("Config generation is not deterministic.\n\n\nFirst generation: \n\n%s\n\nDifferent generation: \n\n%s\n\n", string(cfg), string(testcfg))
		}
	}
}

func generateTestConfig() ([]byte, error) {
	replicas := int32(1)
	return generateConfig(
		&v1alpha1.ElasticsearchCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: v1alpha1.ElasticsearchClusterSpec{
				Replicas: &replicas,
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
		},
	)
}

package elasticsearch

import (
	"reflect"
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

		if !reflect.DeepEqual(cfg, testcfg) {
			t.Fatalf("Config generation is not deterministic.\n\n\nFirst generation: \n\n%+v\n\nDifferent generation: \n\n%+v\n\n", cfg, testcfg)
		}
	}
}

func generateTestConfig() (map[string][]byte, error) {
	replicas := int32(1)
	return generateConfig(
		&v1alpha1.Elasticsearch{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: v1alpha1.ElasticsearchSpec{
				Master: &v1alpha1.ElasticsearchPartSpec{
					Replicas: &replicas,
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("400Mi"),
						},
					},
				},
			},
		},
		"master",
	)
}

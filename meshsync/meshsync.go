package meshsync

import (
	"github.com/meshery/meshkit/broker"
	"github.com/meshery/meshkit/config"
	"github.com/meshery/meshkit/logger"
	mesherykube "github.com/meshery/meshkit/utils/kubernetes"
	"github.com/meshery/meshsync/internal/channels"
	"github.com/meshery/meshsync/internal/output"
	iutils "github.com/meshery/meshsync/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// Handler contains all handlers, channels, clients, and other parameters for an adapter.
// Use type embedding in a specific adapter to extend it.
type Handler struct {
	Config config.Handler
	Log    logger.Handler
	Broker broker.Handler

	clusterID    string
	informer     dynamicinformer.DynamicSharedInformerFactory
	kubeClient   *mesherykube.Client
	channelPool  map[string]channels.GenericChannel
	stores       map[string]cache.Store
	outputWriter output.Writer
}

func GetListOptionsFunc(config config.Handler) (func(*v1.ListOptions), error) {
	var blacklist []string
	err := config.GetObject("spec.informer_config", blacklist)
	if err != nil {
		return nil, err
	}

	return func(lo *v1.ListOptions) {
		// Create a label selector to include all objects
		labelSelector := &v1.LabelSelector{}

		// Add label selector requirements to exclude blacklisted types
		labelSelectorReq := v1.LabelSelectorRequirement{
			Key:      "type",
			Operator: v1.LabelSelectorOpNotIn,
			Values:   blacklist,
		}
		labelSelector.MatchExpressions = append(labelSelector.MatchExpressions, labelSelectorReq)
	}, nil
}

func New(
	config config.Handler,
	kubeClient *mesherykube.Client,
	log logger.Handler,
	br broker.Handler,
	ow output.Writer,
	pool map[string]channels.GenericChannel,
) (*Handler, error) {
	listOptionsFunc, err := GetListOptionsFunc(config)
	if err != nil {
		return nil, err
	}

	clusterID := iutils.GetClusterID(kubeClient.KubeClient)
	informer := GetDynamicInformer(config, kubeClient.DynamicKubeClient, listOptionsFunc)

	return &Handler{
		Config:       config,
		Log:          log,
		Broker:       br,
		outputWriter: ow,
		informer:     informer,
		kubeClient:   kubeClient,
		clusterID:    clusterID,
		channelPool:  pool,
	}, nil
}

func GetDynamicInformer(config config.Handler, dynamicKubeClient dynamic.Interface, listOptionsFunc func(*v1.ListOptions)) dynamicinformer.DynamicSharedInformerFactory {
	return dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicKubeClient, 0, v1.NamespaceAll, listOptionsFunc)
}

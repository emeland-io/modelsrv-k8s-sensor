package sensor

import (
	"fmt"

	"github.com/google/uuid"

	"go.emeland.io/modelsrv/pkg/model"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// K8sSensorNodeTypeID is the stable identity for the "k8s-sensor" NodeType.
// Shared across all instances of this sensor. Must not change after first deployment.
var K8sSensorNodeTypeID = uuid.MustParse("a1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e")

// K8sNamespaceContextTypeID is the stable identity for the "kubernetes-namespace" ContextType.
var K8sNamespaceContextTypeID = uuid.MustParse("c7e8f9a0-b1c2-4d3e-a4f5-6a7b8c9d0e1f")

// Identity holds references to the sensor's own Node so it can be deleted on shutdown.
type Identity struct {
	NodeID uuid.UUID
	Model  model.Model
}

// Register creates the k8s-sensor NodeType, a Node instance, and the
// kubernetes-namespace ContextType in the model. Call Close() on shutdown
// to remove the Node.
func Register(m model.Model) (*Identity, error) {
	nt := node.NewNodeType(K8sSensorNodeTypeID)
	nt.SetDisplayName("k8s-sensor")
	nt.SetDescription("Kubernetes cluster sensor for EmELand")
	if err := m.AddNodeType(nt); err != nil {
		return nil, fmt.Errorf("register node type: %w", err)
	}

	nodeID := uuid.New()
	n := node.NewNode(nodeID)
	n.SetDisplayName("k8s-sensor")
	n.SetNodeTypeByRef(nt)
	if err := m.AddNode(n); err != nil {
		return nil, fmt.Errorf("register node: %w", err)
	}

	ct := mdlctx.NewContextType(K8sNamespaceContextTypeID)
	ct.SetDisplayName("Kubernetes Namespace")
	ct.SetDescription("A Kubernetes namespace mapped as an EmELand context")
	if err := m.AddContextType(ct); err != nil {
		return nil, fmt.Errorf("register context type: %w", err)
	}

	return &Identity{NodeID: nodeID, Model: m}, nil
}

// Close removes the sensor's Node from the model (emits a delete event).
func (id *Identity) Close() error {
	if id == nil {
		return nil
	}
	return id.Model.DeleteNodeById(id.NodeID)
}

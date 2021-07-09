package scheduling

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	clienttesting "k8s.io/client-go/testing"

	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
	clusterapiv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	testinghelpers "open-cluster-management.io/placement/pkg/helpers/testing"
)

func TestSchedule(t *testing.T) {
	clusterSetName := "clusterSets"
	placementNamespace := "ns1"
	placementName := "placement1"

	cases := []struct {
		name            string
		placement       *clusterapiv1alpha1.Placement
		initObjs        []runtime.Object
		clusters        []*clusterapiv1.ManagedCluster
		decisions       []runtime.Object
		scheduleResult  scheduleResult
		validateActions func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:      "new placement satisfied",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
			},
			decisions: []runtime.Object{},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:   1,
				scheduledDecisions: 1,
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				// check if PlacementDecision has been created
				testinghelpers.AssertActions(t, actions, "create", "update")
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, "cluster1")
			},
		},
		{
			name:      "new placement unsatisfied",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(3).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
			},
			decisions: []runtime.Object{},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:     1,
				scheduledDecisions:   1,
				unscheduledDecisions: 2,
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				// check if PlacementDecision has been updated
				testinghelpers.AssertActions(t, actions, "create", "update")
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, "cluster1")
			},
		},
		{
			name:      "placement with all decisions scheduled",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(2).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1", "cluster2").Build(),
			},
			decisions: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1", "cluster2").Build(),
			},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster2").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster3").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:   3,
				scheduledDecisions: 2,
			},
			validateActions: testinghelpers.AssertNoActions,
		},
		{
			name:      "placement with part of decisions scheduled",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(4).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1").Build(),
			},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster2").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			decisions: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1").Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:     2,
				scheduledDecisions:   2,
				unscheduledDecisions: 2,
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				// check if PlacementDecision has been updated
				testinghelpers.AssertActions(t, actions, "update")
				actual := actions[0].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, "cluster1", "cluster2")
			},
		},
		{
			name:      "placement without more feasible cluster available",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(4).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1").Build(),
			},
			decisions: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster1").Build(),
			},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:     1,
				scheduledDecisions:   1,
				unscheduledDecisions: 3,
			},
			validateActions: testinghelpers.AssertNoActions,
		},
		{
			name:      "schedule to cluster with least decisions",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(1).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 1)).
					WithDecisions("cluster1", "cluster2").Build(),
			},
			decisions: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 1)).
					WithDecisions("cluster1", "cluster2").Build(),
			},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster2").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster3").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:   3,
				scheduledDecisions: 1,
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				// check if PlacementDecision has been updated
				testinghelpers.AssertActions(t, actions, "create", "update")
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, "cluster3")
			},
		},
		{
			name:      "do not schedule to other cluster even with least decisions",
			placement: testinghelpers.NewPlacement(placementNamespace, placementName).WithNOC(1).Build(),
			initObjs: []runtime.Object{
				testinghelpers.NewClusterSet(clusterSetName),
				testinghelpers.NewClusterSetBinding(placementNamespace, clusterSetName),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 1)).
					WithDecisions("cluster3", "cluster2").Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 2)).
					WithDecisions("cluster2", "cluster1").Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster3").Build(),
			},
			decisions: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 1)).
					WithDecisions("cluster3", "cluster2").Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName("others", 2)).
					WithDecisions("cluster2", "cluster1").Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions("cluster3").Build(),
			},
			clusters: []*clusterapiv1.ManagedCluster{
				testinghelpers.NewManagedCluster("cluster1").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster2").WithLabel(clusterSetLabel, clusterSetName).Build(),
				testinghelpers.NewManagedCluster("cluster3").WithLabel(clusterSetLabel, clusterSetName).Build(),
			},
			scheduleResult: scheduleResult{
				feasibleClusters:   3,
				scheduledDecisions: 1,
			},
			validateActions: testinghelpers.AssertNoActions,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.initObjs = append(c.initObjs, c.placement)
			clusterClient := clusterfake.NewSimpleClientset(c.initObjs...)
			s := newPluginScheduler(testinghelpers.NewFakePluginHandle(t, clusterClient, c.initObjs...))
			result, err := s.schedule(
				context.TODO(),
				c.placement,
				c.clusters,
			)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}
			if result.feasibleClusters != c.scheduleResult.feasibleClusters {
				t.Errorf("expected %d feasible clusters, but got %d", c.scheduleResult.feasibleClusters, result.feasibleClusters)
			}
			if result.scheduledDecisions != c.scheduleResult.scheduledDecisions {
				t.Errorf("expected %d scheduled, but got %d", c.scheduleResult.scheduledDecisions, result.scheduledDecisions)
			}
			if result.unscheduledDecisions != c.scheduleResult.unscheduledDecisions {
				t.Errorf("expected %d unscheduled, but got %d", c.scheduleResult.unscheduledDecisions, result.unscheduledDecisions)
			}
			c.validateActions(t, clusterClient.Actions())
		})
	}
}

func TestBind(t *testing.T) {
	placementNamespace := "ns1"
	placementName := "placement1"

	cases := []struct {
		name             string
		initObjs         []runtime.Object
		clusterDecisions []clusterapiv1alpha1.ClusterDecision
		validateActions  func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:             "create single placementdecision",
			clusterDecisions: newClusterDecisions(10),
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				testinghelpers.AssertActions(t, actions, "create", "update")
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, newSelectedClusters(10)...)
			},
		},
		{
			name:             "create multiple placementdecisions",
			clusterDecisions: newClusterDecisions(101),
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				testinghelpers.AssertActions(t, actions, "create", "update", "create", "update")
				selectedClusters := newSelectedClusters(101)
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, selectedClusters[0:100]...)

				actual = actions[3].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok = actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, selectedClusters[100:]...)
			},
		},
		{
			name:             "no change",
			clusterDecisions: newClusterDecisions(128),
			initObjs: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[:100]...).Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 2)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[100:]...).Build(),
			},
			validateActions: testinghelpers.AssertNoActions,
		},
		{
			name:             "update one of placementdecisions",
			clusterDecisions: newClusterDecisions(128),
			initObjs: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[:100]...).Build(),
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				testinghelpers.AssertActions(t, actions, "create", "update")
				selectedClusters := newSelectedClusters(128)
				actual := actions[1].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, selectedClusters[100:]...)
			},
		},
		{
			name:             "delete redundant placementdecisions",
			clusterDecisions: newClusterDecisions(10),
			initObjs: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[:100]...).Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 2)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[100:]...).Build(),
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				testinghelpers.AssertActions(t, actions, "update", "delete")
				actual := actions[0].(clienttesting.UpdateActionImpl).Object
				placementDecision, ok := actual.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					t.Errorf("expected PlacementDecision was updated")
				}
				assertClustersSelected(t, placementDecision.Status.Decisions, newSelectedClusters(10)...)
			},
		},
		{
			name:             "delete all placementdecisions",
			clusterDecisions: newClusterDecisions(0),
			initObjs: []runtime.Object{
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 1)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[:100]...).Build(),
				testinghelpers.NewPlacementDecision(placementNamespace, placementDecisionName(placementName, 2)).
					WithLabel(placementLabel, placementName).
					WithDecisions(newSelectedClusters(128)[100:]...).Build(),
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				testinghelpers.AssertActions(t, actions, "delete", "delete")
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clusterClient := clusterfake.NewSimpleClientset(c.initObjs...)

			// GenerateName is not working for fake clent, set the name with random suffix
			clusterClient.PrependReactor(
				"create",
				"placementdecisions",
				func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
					createAction := action.(clienttesting.CreateActionImpl)
					pd := createAction.Object.(*clusterapiv1alpha1.PlacementDecision)
					pd.Name = fmt.Sprintf("%s%s", pd.GenerateName, rand.String(5))
					return false, pd, nil
				},
			)

			s := newPluginScheduler(testinghelpers.NewFakePluginHandle(t, clusterClient, c.initObjs...))

			err := s.bind(
				context.TODO(),
				testinghelpers.NewPlacement(placementNamespace, placementName).Build(),
				c.clusterDecisions,
				map[string]*pluginScore{},
			)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}
			c.validateActions(t, clusterClient.Actions())
		})
	}
}

func assertClustersSelected(t *testing.T, decisons []clusterapiv1alpha1.ClusterDecision, clusterNames ...string) {
	names := sets.NewString(clusterNames...)
	for _, decision := range decisons {
		if names.Has(decision.ClusterName) {
			names.Delete(decision.ClusterName)
		}
	}

	if names.Len() != 0 {
		t.Errorf("expected clusters selected: %s, but got %v", strings.Join(names.UnsortedList(), ","), decisons)
	}
}

func newClusterDecisions(num int) (decisions []clusterapiv1alpha1.ClusterDecision) {
	for i := 0; i < num; i++ {
		decisions = append(decisions, clusterapiv1alpha1.ClusterDecision{
			ClusterName: fmt.Sprintf("cluster%d", i+1),
		})
	}
	return decisions
}

func newSelectedClusters(num int) (clusters []string) {
	for i := 0; i < num; i++ {
		clusters = append(clusters, fmt.Sprintf("cluster%d", i+1))
	}

	sort.SliceStable(clusters, func(i, j int) bool {
		return clusters[i] < clusters[j]
	})

	return clusters
}

func placementDecisionName(placementName string, index int) string {
	return fmt.Sprintf("%s-decision-%d", placementName, index)
}

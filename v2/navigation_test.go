package pluginsdk

import (
	common "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
	"testing"
)

func TestNavigationContractReexports(t *testing.T) {
	var ref *common.NavReference = &NavReference{Owner: "core", ID: "work"}
	item := NavItem{ID: "files", Label: "Files", Href: "/files", Section: ref}
	if err := ValidateNavigation([]NavItem{item}); err != nil {
		t.Fatal(err)
	}
	var snapshot common.NavigationSnapshot = NavigationSnapshot{Items: []NavigationNode{{Owner: "files", Item: item, Section: ref}}}
	if snapshot.Items[0].Section.Owner != "core" {
		t.Fatal("lost owner-qualified section")
	}
	if err := ValidateNavigation([]NavItem{{ID: "bad", Label: "Bad", Kind: NavKindGroup, Href: "/bad"}}); err == nil {
		t.Fatal("structural link accepted")
	}
	if NavigationChildrenInterface != common.NavigationChildrenInterface {
		t.Fatal("interface drift")
	}
}

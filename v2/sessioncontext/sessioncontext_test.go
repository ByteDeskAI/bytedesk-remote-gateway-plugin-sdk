package sessioncontext

import (
	"reflect"
	"testing"
)

func TestCommonContractIsReExportedWithoutGatewayFields(t *testing.T) {
	if ContractRevision != 1 {
		t.Fatalf("revision = %d", ContractRevision)
	}
	if Open.Name() != CommandOpen || Refresh.Name() != CommandRefresh || Action.Name() != CommandAction {
		t.Fatalf("descriptor names = %q, %q, %q", Open.Name(), Refresh.Name(), Action.Name())
	}
	for _, name := range []string{"CWD", "Root", "Port", "Proxy", "Server", "Lease", "Principal"} {
		if _, exists := reflect.TypeOf(Context{}).FieldByName(name); exists {
			t.Fatalf("Context exposes forbidden field %s", name)
		}
	}
}

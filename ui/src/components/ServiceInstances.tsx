import * as ui5 from "@ui5/webcomponents-react";

function ServiceInstances() {
  return (
    <>
      <ui5.Table
        columns={
          <>
            <ui5.TableColumn>
              <ui5.Label>Service Instance</ui5.Label>
            </ui5.TableColumn>
          </>
        }
        onLoadMore={function _a() {}}
        onPopinChange={function _a() {}}
        onRowClick={function _a() {
          console.log("row clicked");
        }}
        onSelectionChange={function _a() {}}
      >
        <ui5.TableRow>
          <ui5.TableCell>
            <ui5.Label>service-instance-1</ui5.Label>
          </ui5.TableCell>
        </ui5.TableRow>
        <ui5.TableRow>
          <ui5.TableCell>
            <ui5.Label>service-instance-2</ui5.Label>
          </ui5.TableCell>
        </ui5.TableRow>
        <ui5.TableRow>
          <ui5.TableCell>
            <ui5.Label>service-instance-3</ui5.Label>
          </ui5.TableCell>
        </ui5.TableRow>
        <ui5.TableRow>
          <ui5.TableCell>
            <ui5.Label>service-instance-4</ui5.Label>
          </ui5.TableCell>
        </ui5.TableRow>
      </ui5.Table>
    </>
  );
}

export default ServiceInstances;

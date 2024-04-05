import * as ui5 from "@ui5/webcomponents-react";
import { ServiceInstanceBrief, ServiceInstancesBrief } from "../shared/models";
import axios from "axios";
import { useEffect, useState } from "react";

function ServiceInstances() {
  const [serviceInstancesBrief, setServiceInstancesBrief] = useState<ServiceInstancesBrief>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  
  useEffect(() => {
    axios
      .get<ServiceInstancesBrief>(
        `http://localhost:3002/api/list-service-instances`
      )
      .then((response) => {
        setServiceInstancesBrief(response.data);
        setLoading(false);
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <ui5.Text>Loading...</ui5.Text>;
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  const renderData = () => {
    return serviceInstancesBrief?.items.map((brief, index) => {
      return (
        <>
        <ui5.TableRow>
          <ui5.TableCell>
            <ui5.Label>{brief.name}</ui5.Label>
          </ui5.TableCell>
          <ui5.TableCell>
            <ui5.Label>{brief.namespace}</ui5.Label>
          </ui5.TableCell>
          <ui5.TableCell>
            <ui5.Label>{brief.context.join("&")}</ui5.Label>
          </ui5.TableCell>
          <ui5.TableCell>
            <ui5.Label>{brief.service_bindings.length}</ui5.Label>
          </ui5.TableCell>
        </ui5.TableRow>
        </>
      );
    });
  };


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
        {renderData()}
      </ui5.Table>
    </>
  );
}
function format(s :ServiceInstanceBrief) {
  const sbLen = s.service_bindings.length
  return `${s.name} in ${s.namespace} on ${s.context} with ${sbLen} bindings`
}
export default ServiceInstances;

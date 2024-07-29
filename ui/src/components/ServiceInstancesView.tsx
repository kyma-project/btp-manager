import * as ui5 from "@ui5/webcomponents-react";
import { ServiceInstances } from "../shared/models";
import axios from "axios";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import api from "../shared/api";
import Ok from "../shared/validator";
import serviceInstancesData from '../test-data/serivce-instances.json';
import ServiceInstancesDetailsView from "./ServiceInstancesDetailsView";

function ServiceInstancesView() {
  const [serviceInstances, setServiceInstances] = useState<ServiceInstances>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [portal, setPortal] = useState<JSX.Element>();

  useEffect(() => {
    var useTestData = process.env.REACT_APP_USE_TEST_DATA === "true"
    if (!useTestData) {
      setLoading(true)
      axios
        .get<ServiceInstances>(api("service-instances"))
        .then((response) => {
          setLoading(false);
          setServiceInstances(response.data);
        })
        .catch((error) => {
          setLoading(false);
          setError(error);
        });
      setLoading(false)
    } else {
      setLoading(true)
      setServiceInstances(serviceInstancesData)
      setLoading(false);
    }
  }, []);

  if (loading) {
    return <ui5.Loader progress="100%" />
  }

  if (error) {
    return <ui5.IllustratedMessage name="UnableToLoad" />
  }

  const renderData = () => {
    // @ts-ignore
    if (!Ok(serviceInstances) || !Ok(serviceInstances.items)) {
      return <ui5.IllustratedMessage name="NoEntries" />
    }
    return serviceInstances?.items.map((instance, index) => {
      return (
        <>
          <ui5.TableRow 
          onClick={() => {
            console.log("Row clicked")
            const instanceView = <ServiceInstancesDetailsView 
            key={instance.id} instance={instance} open={true} />
            const portal = createPortal( instanceView, document.getElementById("App")!!)
            setPortal(portal)
          }}
        >
            
            <ui5.TableCell>
              <ui5.Label>{instance.name}</ui5.Label>
            </ui5.TableCell>

            <ui5.TableCell>
              <ui5.Label>{instance.namespace}</ui5.Label>
            </ui5.TableCell>

          </ui5.TableRow>
        </>
      );
    });
  };

  return (
    <>
    {
      <ui5.Card>

        <ui5.Table
          columns={
            <>
              <ui5.TableColumn>
                <ui5.Label>Service Instance</ui5.Label>
              </ui5.TableColumn>

              <ui5.TableColumn>
                <ui5.Label>Service Namespace</ui5.Label>
              </ui5.TableColumn>
            </>
          }
        >
          {renderData()}
        </ui5.Table>
      </ui5.Card>

    }
    {portal != null && portal}

    </>
  );
}

export default ServiceInstancesView;
import * as ui5 from "@ui5/webcomponents-react";
import { ServiceInstance, ServiceInstances } from "../shared/models";
import axios from "axios";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import api from "../shared/api";
import Ok from "../shared/validator";
import serviceInstancesData from '../test-data/serivce-instances.json';
import ServiceInstancesDetailsView from "./ServiceInstancesDetailsView";
import { useParams } from "react-router-dom";
import StatusMessage from "./StatusMessage";

function ServiceInstancesView() {
  const [serviceInstances, setServiceInstances] = useState<ServiceInstances>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedInstance, setSelectedInstance] = useState<ServiceInstance>(new ServiceInstance());
  const dialogRef = useRef();
  const [success, setSuccess] = useState("");

  let { id } = useParams();

  useEffect(() => {
    var useTestData = process.env.REACT_APP_USE_TEST_DATA === "true"
    if (!useTestData) {
      setLoading(true)
      axios
        .get<ServiceInstances>(api("service-instances"))
        .then((response) => {
          setLoading(false);
          setServiceInstances(response.data);
          if (id) {
            const instance = response.data.items.find((instance) => instance.id === id);
            if (instance) {
              openPortal(instance)
            }
          }
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
  }, [id]);

  if (loading) {
    return <ui5.BusyIndicator
      active
      delay={1000}
      size="Medium"
    />
  }

  function openPortal(instance: any) {
    setSelectedInstance(instance)
    //@ts-ignore
    dialogRef.current.open()
  }

  function deleteInstance(id: string): boolean {
    setLoading(true);
    axios
      .delete(api("service-instances") + "/" + id)
      .then((response) => {
        serviceInstances!!.items = serviceInstances!!.items.filter(instance => instance.id !== id)
        setServiceInstances(serviceInstances);
        setSuccess("Service instance deleted successfully")
        setError(null)
        setLoading(false);

      })
      .catch((error) => {
        setLoading(false);
        setError(error);
      });

    return true;
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
            selected={id === instance.id}
            onClick={() => {
              openPortal(instance)
            }}
          >

            <ui5.TableCell>
              <ui5.Label>{instance.name}</ui5.Label>
            </ui5.TableCell>

            <ui5.TableCell>
              <ui5.Label>{instance.namespace}</ui5.Label>
            </ui5.TableCell>

            <ui5.TableCell>
              <ui5.Label>
                <ui5.Button
                  design="Default"
                  icon="delete"
                  onClick={function _a(e: any) {
                    e.preventDefault();
                    e.stopPropagation();
                    deleteInstance(instance.id);
                    return true;
                  }}
                >
                </ui5.Button>
              </ui5.Label>
            </ui5.TableCell>


          </ui5.TableRow>
        </>
      );
    });
  };

  return (
    <>

        <StatusMessage 
         error={error ?? undefined} success={success} />

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

              <ui5.TableColumn>
                <ui5.Label>Action</ui5.Label>
              </ui5.TableColumn>
            </>
          }
        >
          {renderData()}
        </ui5.Table>
      </ui5.Card>
      {createPortal(<ServiceInstancesDetailsView instance={selectedInstance} ref={dialogRef} />, document.getElementById("App")!!)}
    </>
  );
}

export default ServiceInstancesView;
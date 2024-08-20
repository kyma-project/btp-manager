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
import {ServiceOfferings} from "../shared/models";

function ServiceInstancesView(props: any) {
  const [serviceInstances, setServiceInstances] = useState<ServiceInstances>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedInstance, setSelectedInstance] = useState<ServiceInstance>(new ServiceInstance());
  const dialogRef = useRef();
  const [success, setSuccess] = useState("");
  const [, setOfferings] = useState<ServiceOfferings>();

  let { id } = useParams();

  useEffect(() => {
    setLoading(true)
    if (!Ok(props.setTitle)) {
      return;
    }
    props.setTitle("Service Instances");

    var useTestData = process.env.REACT_APP_USE_TEST_DATA === "true"
    if (!useTestData) {

      if (!Ok(props.secret)) {
        return;
      }
      const secretText = splitSecret(props.secret);
      if (Ok(secretText)) {
        axios
          .get<ServiceOfferings>(
            api(`service-offerings/${secretText.namespace}/${secretText.secret_name}`)
          )
          .then((response) => {
            setOfferings(response.data);
            axios
              .get<ServiceInstances>(api("service-instances"))
              .then((response) => {
                setServiceInstances(response.data);
                setError(null);
                if (id) {
                  const instance = response.data.items.find((instance) => instance.id === id);
                  if (instance) {
                    openPortal(instance);
                  }
                }
                setLoading(false);
              })
              .catch((error) => {
                setLoading(false);
                setError(error);
              });
          })
          .catch((error) => {
            setLoading(false);
            setError(error);
          });
      } 
    } else {
      setLoading(true)
      setServiceInstances(serviceInstancesData)
      setLoading(false);
    }
  }, [id, props, props.secret]);

    
  if (loading) {
    return <ui5.BusyIndicator
      active
      delay={1}
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
    if (error) {
      return <ui5.IllustratedMessage name="NoEntries"/>
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

function splitSecret(secret: string) {
  if (secret == null) {
      return {};
  }
  const secretParts = secret.split(" ");
  const secret_name = secretParts[0];
  let namespace = secretParts[2].replace("(", "");
  namespace = namespace.replace(")", "");
  return {secret_name, namespace};
}

export default ServiceInstancesView;
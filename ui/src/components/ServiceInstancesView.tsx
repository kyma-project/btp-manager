import * as ui5 from "@ui5/webcomponents-react";
import { Secret, ServiceInstance, ServiceInstances } from "../shared/models";
import axios from "axios";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import api from "../shared/api";
import Ok from "../shared/validator";
import serviceInstancesData from '../test-data/serivce-instances.json';
import ServiceInstancesDetailsView from "./ServiceInstancesDetailsView";
import { useParams } from "react-router-dom";
import StatusMessage from "./StatusMessage";
import { splitSecret } from "../shared/common";
import { FCLLayout, FlexibleColumnLayout } from "@ui5/webcomponents-react";

function ServiceInstancesView(props: any) {
  const [serviceInstances, setServiceInstances] = useState<ServiceInstances>();
  const [secret, setSecret] = useState<Secret>(new Secret());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedInstance, setSelectedInstance] = useState<ServiceInstance>(new ServiceInstance());
  const [success, setSuccess] = useState("");
  const [layout, setLayout] = useState(FCLLayout.OneColumn);

  let { id } = useParams();

  useEffect(() => {
    setLoading(true)

    // disable selection when page refresh is done
    setSelectedInstance(new ServiceInstance());

    if (!Ok(props.setTitle)) {
      return;
    }
    props.setTitle("Service Instances");

    var useTestData = process.env.REACT_APP_USE_TEST_DATA === "true"
    if (!useTestData) {

      if (!Ok(props.secret)) {
        return;
      }
      const secret = splitSecret(props.secret);
      setSecret(secret);
      if (Ok(secret)) {
        axios
          .get<ServiceInstances>(api("service-instances"), {
            params: {
              sm_secret_name: secret.name,
              sm_secret_namespace: secret.namespace
            }
          })
          .then((response) => {
            setServiceInstances(response.data);
            setError(null);
            if (id) {
              const instance = response.data.items.find((instance) => instance.id === id);
              if (instance) {
                setSelectedInstance(instance)
                setLayout(FCLLayout.TwoColumnsMidExpanded)
              }
            }
            setLoading(false);
          })
          .catch((error) => {
            setServiceInstances(undefined);
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

  function deleteInstance(id: string): boolean {
    setLoading(true);
    axios
      .delete(api("service-instances") + "/" + id, {
        params: {
          sm_secret_name: secret.name,
          sm_secret_namespace: secret.namespace
        }
      })
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
              setSelectedInstance(instance)
              setLayout(FCLLayout.TwoColumnsMidExpanded)
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
      <FlexibleColumnLayout id="fcl" layout={layout}>

      <div  selection-mode="Single" slot="startColumn" className="margin-wrapper">

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

      </div>


        <div slot="midColumn" >
          <ui5.Bar>
            <div className="icons-container" slot="endContent">
              <ui5.Button design="Transparent" icon="full-screen" onClick={() => {
                if (layout === FCLLayout.MidColumnFullScreen) {
                  setLayout(FCLLayout.TwoColumnsMidExpanded)
                } else {
                  setLayout(FCLLayout.MidColumnFullScreen)
                }
              }}></ui5.Button>
              <ui5.Button icon="decline" design="Transparent" onClick={() => {
                setLayout(FCLLayout.OneColumn)
              }}></ui5.Button>
            </div>
          </ui5.Bar>
          <ServiceInstancesDetailsView secret={secret} instance={selectedInstance} />
        </div>

      </FlexibleColumnLayout>

    </>
  );
}



export default ServiceInstancesView;
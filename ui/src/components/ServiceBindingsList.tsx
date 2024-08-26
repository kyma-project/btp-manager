import * as ui5 from "@ui5/webcomponents-react";
import { ApiError, ServiceInstanceBinding, ServiceInstanceBindings } from "../shared/models";
import axios from "axios";
import { forwardRef, useEffect, useImperativeHandle, useState } from "react";
import api from "../shared/api";
import Ok from "../shared/validator";
import serviceInstancesData from '../test-data/service-bindings.json';
import StatusMessage from "./StatusMessage";

const ServiceBindingsList = forwardRef((props: any, ref) => {
  const [bindings, setServiceInstanceBindings] = useState<ServiceInstanceBindings>(new ServiceInstanceBindings());

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError>();
  const [portal] = useState<JSX.Element>();
  const [success] = useState("");

  useImperativeHandle(ref, () => ({

    add(binding: ServiceInstanceBinding) {
      bindings?.items.push(binding);
      console.log(bindings)
      const newbindings = new ServiceInstanceBindings();
      newbindings.items = bindings?.items ?? [];
      setServiceInstanceBindings(newbindings);
    }

  }));

  function deleteBinding(id: string): boolean {
    setLoading(true);

    axios
      .delete(api("service-bindings") + "/" + id, {
        params: {
          sm_secret_name: props.secret.name,
          sm_secret_namespace: props.secret.namespace
        }
      })
      .then((response) => {
        bindings!!.items = bindings!!.items.filter(instance => instance.id !== id)
        setServiceInstanceBindings(bindings);
        setLoading(false);
        setError(undefined);
      })
      .catch((error) => {
        setLoading(false);
        setError(error);
      });

    return true;
  }

  useEffect(() => {
    setLoading(true)

    if (!Ok(props.secret) || !Ok(props.secret.name) || !Ok(props.secret.namespace)) {
      setServiceInstanceBindings(new ServiceInstanceBindings());
      return;
    }

    if (!Ok(props.instance)) {
      setServiceInstanceBindings(new ServiceInstanceBindings());
      return;
    }

    if (!Ok(props.instance.id)) {
      setServiceInstanceBindings(new ServiceInstanceBindings());
      return;
    }
  
    var useTestData = process.env.REACT_APP_USE_TEST_DATA === "true"
    if (!useTestData) {
      axios
        .get<ServiceInstanceBindings>(api("service-bindings"),
          {
            params:
            {
              service_instance_id: props.instance.id,
              sm_secret_name: props.secret.name,
              sm_secret_namespace: props.secret.namespace
            }
          }
        )
        .then((response) => {
          if (Ok(response.data)) {
            setServiceInstanceBindings(response.data);
          } else {
            setServiceInstanceBindings(new ServiceInstanceBindings());
          }
          setError(undefined);
          setLoading(false);
        })
        .catch((error) => {
          setError(error);
          setLoading(false);
        });
      setLoading(false)
    } else {
      setServiceInstanceBindings(serviceInstancesData)
      setLoading(false);
    }
  }, [props.instance, props.instance.id, props.secret]);

  if (loading) {
    return <ui5.BusyIndicator
      active
      delay={1}
      size="Medium"
    />
  }

  const renderData = () => {
    // @ts-ignore
    if (!Ok(bindings) || !Ok(bindings.items)) {
      return <ui5.IllustratedMessage name="NoEntries" />
    }
    return bindings?.items.map((binding, index) => {
      return (
        <ui5.TableRow>

          <ui5.TableCell>
            <ui5.Label>{binding.id}</ui5.Label>
          </ui5.TableCell>

          <ui5.TableCell>
            <ui5.Label>{binding.name}</ui5.Label>
          </ui5.TableCell>

          <ui5.TableCell>
            <ui5.Label>{binding.secret_name}</ui5.Label>
          </ui5.TableCell>

          <ui5.TableCell>
            <ui5.Label>{binding.secret_namespace}</ui5.Label>
          </ui5.TableCell>

          <ui5.TableCell>
            <ui5.Button
              design="Default"
              icon="delete"
              onClick={function _a(e: any) {
                e.stopPropagation();
                return deleteBinding(binding.id);
              }}
            >
            </ui5.Button>
          </ui5.TableCell>

        </ui5.TableRow>
      );
    });
  };

  if (!Ok(bindings) || !Ok(bindings.items)) {
    return <ui5.IllustratedMessage name="NoEntries" size="Dot" />
  }

  return (
    <>
      <ui5.Form>
        <StatusMessage error={error ?? undefined} success={success} />
      </ui5.Form>

      {

        <ui5.Table
          columns={
            <>
              <ui5.TableColumn>
                <ui5.Label>Id</ui5.Label>
              </ui5.TableColumn>

              <ui5.TableColumn>
                <ui5.Label>Name</ui5.Label>
              </ui5.TableColumn>

              <ui5.TableColumn>
                <ui5.Label>Secret Name</ui5.Label>
              </ui5.TableColumn>

              <ui5.TableColumn>
                <ui5.Label>Secret Namespace</ui5.Label>
              </ui5.TableColumn>

              <ui5.TableColumn>
                <ui5.Label>Action</ui5.Label>
              </ui5.TableColumn>
            </>
          }
        >
          {renderData()}
        </ui5.Table>
      }

      {portal != null && portal}

    </>
  );
});

export default ServiceBindingsList;
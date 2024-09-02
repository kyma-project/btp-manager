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
    },

    refresh(binding: ServiceInstanceBinding) {
      const newbindings = new ServiceInstanceBindings();
      newbindings.items = bindings?.items ?? [];
      let bindingIndex = newbindings!!.items.findIndex(bindingInArray => bindingInArray.id === binding.id)
      let foundBinding = newbindings.items[bindingIndex]
      foundBinding!!.secret_name = binding.secret_name
      foundBinding!!.secret_namespace = binding.secret_namespace
      newbindings.items[bindingIndex] = foundBinding
      console.log(newbindings)
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

    function toggleSecretRestore(sb: ServiceInstanceBinding, buttonPressed: boolean): boolean {
      props.setSecretRestoreButtonPressedState(buttonPressed);
      if (buttonPressed) {
            props.setServiceBinding(sb);
        } else {
            props.setServiceBinding(new ServiceInstanceBinding());
      }
        return true;
    }

  function sbTableRow(sb: ServiceInstanceBinding) {
      let buttons;
      if (!Ok(sb.secret_name) || !Ok(sb.secret_namespace)) {
          buttons =
              <ui5.TableCell>

                  <ui5.Button
                      design="Default"
                      icon="delete"
                      tooltip="Delete Service Binding"
                      onClick={function _a(e: any) {
                          e.stopPropagation();
                          return deleteBinding(sb.id);
                      }}
                  />

                  <ui5.ToggleButton
                      design="Default"
                      icon="synchronize"
                      tooltip="Restore Secret"
                      onClick={function _a(e: any) {
                          e.stopPropagation();
                          return toggleSecretRestore(sb, e.target.pressed);
                      }}
                  />

              </ui5.TableCell>
      } else {
            buttons =
                <ui5.TableCell>

                    <ui5.Button
                        design="Default"
                        icon="delete"
                        tooltip="Delete Service Binding"
                        onClick={function _a(e: any) {
                            e.stopPropagation();
                            return deleteBinding(sb.id);
                        }}
                    />

                </ui5.TableCell>
      }
      return (
          <ui5.TableRow>

              <ui5.TableCell>
                  <ui5.Label>{sb.id}</ui5.Label>
              </ui5.TableCell>

              <ui5.TableCell>
                  <ui5.Label>{sb.name}</ui5.Label>
              </ui5.TableCell>

              <ui5.TableCell>
                  <ui5.Label>{sb.secret_name}</ui5.Label>
              </ui5.TableCell>

              <ui5.TableCell>
                  <ui5.Label>{sb.secret_namespace}</ui5.Label>
              </ui5.TableCell>

              {buttons}

        </ui5.TableRow>
    );
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
      return sbTableRow(binding);
    });
  };

  if (!Ok(bindings) || !Ok(bindings.items)) {
    return <ui5.IllustratedMessage name="NoEntries" size="Dot" />
  }

  return (
      <>
        <ui5.Form>
          <StatusMessage error={error ?? undefined} success={success}/>
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
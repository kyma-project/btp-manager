import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  ApiError,
  ServiceInstanceBinding,
} from "../shared/models";
import { useEffect, useState } from "react";
import api from "../shared/api";
import axios from "axios";
import StatusMessage from "./StatusMessage";
import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';
import { generateRandom5CharString } from "../shared/common";

function CreateBindingForm(props: any) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError>();

  const [createdBinding, setCreatedBinding] = useState<ServiceInstanceBinding>(new ServiceInstanceBinding());
  const [success, setSuccess] = useState("");

  function resetCreateBindingForm(response: any) {
      const suffix = "-" + generateRandom5CharString()

      const binding = new ServiceInstanceBinding()
      binding.name = props.instanceName
      binding.secret_name = props.instanceName + suffix
      binding.secret_namespace = "default"

      setCreatedBinding(binding);
      setError(undefined);
      setLoading(false);
      return response;
  }

  const handleCreate = (e: any): boolean => {
    setLoading(true)
    e.preventDefault();
    e.stopPropagation();

    if (e.nativeEvent.submitter.localName === "ui5-multi-input") {
      e.preventDefault()
      return false;
    }

    setLoading(true)
      axios
          .post<ServiceInstanceBinding>(api("service-bindings"), {
              name: createdBinding.name,
              service_instance_id: props.instanceId ?? "",
              secret_name: createdBinding.secret_name,
              secret_namespace: createdBinding.secret_namespace
          }, {
              params:
                  {
                      sm_secret_name: props.secret!!.name,
                      sm_secret_namespace: props.secret!!.namespace
                  }
          })
          .then(resetCreateBindingForm)
          .then((response) => {
              props.onCreate(response.data);
              setSuccess("Item with id " + response.data.name + " created.");
          })
          .catch((error: ApiError) => {
              setLoading(false);
              setError(error);
              setSuccess("");
          });

    e.preventDefault();
    e.stopPropagation();
    return false;
  }

    const handleSecretRestore = (e: any): boolean => {
        e.preventDefault();
        if (e.nativeEvent.submitter.localName === "ui5-multi-input") {
            e.preventDefault()
            return false;
        }

        setLoading(true)
        axios
            .put(api("service-bindings") + "/" + createdBinding.id, {
                name: createdBinding.name,
                service_instance_id: props.instanceId ?? "",
                secret_name: createdBinding.secret_name,
                secret_namespace: createdBinding.secret_namespace
            }, {
                params: {
                    sm_secret_name: props.secret.name,
                    sm_secret_namespace: props.secret.namespace
                }
            })
            .then(resetCreateBindingForm)
            .then((response) => {
                props.onSecretRestore(createdBinding);
                setSuccess("Item with id " + response.data.name + " updated.");
            })
            .catch((error: ApiError) => {
                setLoading(false);
                setError(error);
                setSuccess("");
            });

        e.preventDefault();
        return false;
    }

  useEffect(() => {
    setLoading(true)
    if (!Ok(props.instanceId)) {
      return;
    }

    if (!Ok(props.onCreate)) {
      return;
    }

    if (!Ok(props.secret) || !Ok(props.secret.name) || !Ok(props.secret.namespace)) {
      return;
    }

    setLoading(false);
    setError(undefined);
      const suffix = "-" + generateRandom5CharString()
      const currentBinding = new ServiceInstanceBinding()
    if (props.buttonPressed) {
        currentBinding.id = props.binding.id
        currentBinding.name = props.binding.name
        currentBinding.secret_name = props.binding.name + suffix
        currentBinding.secret_namespace = "default"
      setCreatedBinding(currentBinding)
    } else {
        currentBinding.name = props.instanceName
        currentBinding.secret_name = props.instanceName + suffix
        currentBinding.secret_namespace = "default"
      setCreatedBinding(currentBinding)
    }

  }, [props.instanceId, props.instanceName, props.onCreate, props.secret, props.binding, props.buttonPressed]);

  const renderData = () => {

    if (loading) {
      return <ui5.BusyIndicator
        active
        delay={1}
        size="Medium"
      />
    }

    if (props.buttonPressed) {
        return (
            <ui5.Form
                onSubmit={handleSecretRestore}>
                <ui5.FormItem>
                    <StatusMessage error={error ?? undefined} success={success} />
                </ui5.FormItem>
                <ui5.FormItem label={<ui5.Label required>Name</ui5.Label>}>
                    <ui5.Input
                        style={{ width: "100%" }}
                        required
                        value={createdBinding?.name ?? ''}
                        disabled={true}
                        onChange={(e) => {
                            createdBinding!!.name = e.target.value
                            setCreatedBinding(createdBinding)
                        }}
                    />
                </ui5.FormItem>

                <ui5.FormItem label={<ui5.Label required>Secret Name</ui5.Label>}>
                    <ui5.Input
                        style={{ width: "100%" }}
                        required
                        value={createdBinding?.secret_name ?? ''}
                        onChange={(e) => { // defaulted to service instance name
                            createdBinding!!.secret_name = e.target.value
                            setCreatedBinding(createdBinding)
                        }}
                    />
                </ui5.FormItem>

                <ui5.FormItem label={<ui5.Label required>Secret Namespace</ui5.Label>}>
                    <ui5.Input
                        style={{ width: "100%" }}
                        required // default to "default"
                        value={createdBinding?.secret_namespace ?? ''}
                        onChange={(e) => {
                            createdBinding!!.secret_namespace = e.target.value
                            setCreatedBinding(createdBinding)
                        }}
                    />
                </ui5.FormItem>

                <ui5.FormItem>
                    <ui5.Button type={ui5.ButtonType.Submit}>Submit</ui5.Button>
                </ui5.FormItem>
            </ui5.Form>

        )
    }

    return (
      <ui5.Form
        onSubmit={handleCreate}>
        <ui5.FormItem>
          <StatusMessage error={error ?? undefined} success={success} />
        </ui5.FormItem>
        <ui5.FormItem label={<ui5.Label required>Name</ui5.Label>}>
          <ui5.Input
            style={{ width: "100%" }}
            required
            value={createdBinding?.name ?? ''}
            onChange={(e) => {
              createdBinding!!.name = e.target.value
              setCreatedBinding(createdBinding)
            }}
          />
        </ui5.FormItem>

        <ui5.FormItem label={<ui5.Label required>Secret Name</ui5.Label>}>
          <ui5.Input
            style={{ width: "100%" }}
            required
            value={createdBinding?.secret_name ?? ''}
            onChange={(e) => { // defaulted to service instance name
              createdBinding!!.secret_name = e.target.value
              setCreatedBinding(createdBinding)
            }}
          />
        </ui5.FormItem>

        <ui5.FormItem label={<ui5.Label required>Secret Namespace</ui5.Label>}>
          <ui5.Input
            style={{ width: "100%" }}
            required // default to "default"
            value={createdBinding?.secret_namespace ?? ''}
            onChange={(e) => {
              createdBinding!!.secret_namespace = e.target.value
              setCreatedBinding(createdBinding)
            }}
          />
        </ui5.FormItem>

        <ui5.FormItem>
          <ui5.Button type={ui5.ButtonType.Submit}>Submit</ui5.Button>
        </ui5.FormItem>
      </ui5.Form>

    )
  }
  // @ts-ignore
  return <>{renderData()}</>;
}

export default CreateBindingForm;
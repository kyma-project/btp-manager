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

function CreateBindingForm(props: any) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError>();

  const [createdBinding, setCreatedBinding] = useState<ServiceInstanceBinding>(new ServiceInstanceBinding());
  const [success, setSuccess] = useState("");

  const handleCreate = (e: any): boolean => {
    e.preventDefault();
    e.stopPropagation();

    if (e.nativeEvent.submitter.localName === "ui5-multi-input") {
      e.preventDefault()
      return false;
    }

    createdBinding.serviceInstanceId = props.instanceId ?? ""

    setLoading(true)
    axios
      .post<ServiceInstanceBinding>(api("service-bindings"), {
        name: createdBinding.name,
        service_instance_id: createdBinding.serviceInstanceId,
        secret_name: createdBinding.secretName,
        secret_namespace: createdBinding.secretNamespace
      })
      .then((response) => {
        
        // propagate the created binding
        props.onCreate(response.data);
        
        // reset binding
        const binding = new ServiceInstanceBinding()
        binding.name = props.instanceName
        binding.secretName = props.instanceName
        binding.secretNamespace = "default"

        setSuccess("Item with id " + response.data.name + " created.");
        setCreatedBinding(binding);
        setError(undefined);
        setLoading(false);
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

  useEffect(() => {
    if (!Ok(props.instanceId)) {
      return;
    }

    if (!Ok(props.onCreate)) {
      return;
    }

    setLoading(true)

    setLoading(false)
    setError(undefined)

    createdBinding.name = props.instanceName
    createdBinding.secretName = props.instanceName
    createdBinding.secretNamespace = "default"
    setCreatedBinding(createdBinding)

  }, [createdBinding, props.instanceId, props.instanceName, props.onCreate]);

  const renderData = () => {

    if (loading) {
      return <ui5.BusyIndicator
        active
        delay={1000}
        size="Medium"
      />
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
            value={createdBinding?.secretName ?? ''}
            onChange={(e) => { // defaulted to service instance name
              createdBinding!!.secretName = e.target.value
              setCreatedBinding(createdBinding)
            }}
          />
        </ui5.FormItem>

        <ui5.FormItem label={<ui5.Label required>Secret Namespace</ui5.Label>}>
          <ui5.Input
            style={{ width: "100%" }}
            required // default to "default"
            value={createdBinding?.secretNamespace ?? ''}
            onChange={(e) => {
              createdBinding!!.secretNamespace = e.target.value
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
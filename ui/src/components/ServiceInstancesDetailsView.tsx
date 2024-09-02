import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  ApiError,
  ServiceInstance,
  ServiceInstanceBinding,
} from "../shared/models";
import { forwardRef, useEffect, useRef, useState } from "react";
import ServiceBindingsList from "./ServiceBindingsList";
import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';
import CreateBindingForm from "./CreateBindingForm";

const ServiceInstancesDetailsView = forwardRef((props: any, ref) => {
  const [loading, setLoading] = useState(true);
  const [secret, setSecret] = useState();
  const [error] = useState<ApiError>();

  const [instance, setInstance] = useState<ServiceInstance>();
  const [binding, setBinding] = useState<ServiceInstanceBinding>(new ServiceInstanceBinding());
  const [secretRestoreButtonPressed, setSecretRestoreButtonPressed] = useState(false);
  const listRef = useRef(null);


  const onBindingAdded = (binding: ServiceInstanceBinding) => {
    // @ts-ignore
    listRef.current.add(binding)
  }

  function setServiceBinding(sb: ServiceInstanceBinding) {
    setBinding(sb);
  }

  function setSecretRestoreButtonPressedState(pressed: boolean) {
    setSecretRestoreButtonPressed(pressed)
  }

  function onSecretRestore(sb: ServiceInstanceBinding) {
    // @ts-ignore
    listRef.current.refresh(sb)
  }

  useEffect(() => {
    setLoading(true);
    if (!Ok(props.instance)) {
      return;
    }

    if (!Ok(props.secret) || !Ok(props.secret.name) || !Ok(props.secret.namespace)) {
      return;
    }

    setSecret(props.secret);
    setInstance(props.instance);

    setLoading(false)

  }, [props.instance, props.secret]);

  const renderData = () => {

    if (loading) {
      return <ui5.BusyIndicator
        active
        delay={1}
        size="Medium"
      />
    }

    if (error) {
      return <ui5.IllustratedMessage name="UnableToLoad" />
    }

    return (
    <>
        <ui5.Panel headerLevel="H2" headerText="Service Details">
          <ui5.Form>
            
            <ui5.FormItem label="ID">
              <ui5.Text>{instance?.id}</ui5.Text>
            </ui5.FormItem>

            <ui5.FormItem label="Name">
              <ui5.Text>{instance?.name}</ui5.Text>
            </ui5.FormItem>
          </ui5.Form>
        </ui5.Panel>

        <ui5.Panel accessibleRole="Form" headerLevel="H2" headerText="Bindings">
          <ServiceBindingsList secret={secret} ref={listRef} instance={props.instance} setServiceBinding={setServiceBinding} setSecretRestoreButtonPressedState={setSecretRestoreButtonPressedState} />
        </ui5.Panel>

        <ui5.Panel headerLevel="H2" headerText="Create Binding">
          <CreateBindingForm secret={secret} binding={binding} onCreate={(binding: ServiceInstanceBinding) => onBindingAdded(binding)} instanceId={props.instance.id} instanceName={props.instance.name} buttonPressed={secretRestoreButtonPressed} onSecretRestore={(binding: ServiceInstanceBinding) => onSecretRestore(binding)} />
        </ui5.Panel>
    </>

    )
  }
  // @ts-ignore
  return <>{renderData()}</>;
})

export default ServiceInstancesDetailsView;
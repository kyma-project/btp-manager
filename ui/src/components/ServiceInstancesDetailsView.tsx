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
  const listRef = useRef(null);


  const onBindingAdded = (binding: ServiceInstanceBinding) => {
    // @ts-ignore
    listRef.current.add(binding)
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
          <ServiceBindingsList secret={secret} ref={listRef} instance={props.instance} />
        </ui5.Panel>

        <ui5.Panel headerLevel="H2" headerText="Create Binding">
          <CreateBindingForm secret={secret} onCreate={(binding: ServiceInstanceBinding) => onBindingAdded(binding)} instanceId={props.instance.id} instanceName={props.instance.name} />
        </ui5.Panel>
    </>

    )
  }
  // @ts-ignore
  return <>{renderData()}</>;
})

export default ServiceInstancesDetailsView;
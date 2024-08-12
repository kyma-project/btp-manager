import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  ApiError,
  ServiceInstance,
  ServiceInstanceBinding,
} from "../shared/models";
import { useEffect, useRef, useState } from "react";
import ServiceBindingForm from "./ServiceBindingForm";

const ServiceInstancesDetailsView = forwardRef((props: any, ref) => {
  const [loading, setLoading] = useState(true);
  const [error] = useState<ApiError>();

  const [instance, setInstance] = useState<ServiceInstance>();
  const dialogRef = useRef(null);
  const listRef = useRef(null);

  useImperativeHandle(ref, () => ({

    open() {
      if (dialogRef.current) {
        // @ts-ignore
        dialogRef.current.show();
      }
    }

  }));

  const handleClose = () => {
    if (dialogRef.current) {
      // @ts-ignore
      dialogRef.current.close();
    }
  };

  const onBindingAdded = (binding: ServiceInstanceBinding) => {
    // @ts-ignore
    listRef.current.add(binding)
  }

  useEffect(() => {
    if (!Ok(props.instance)) {
      return;
    }

    setInstance(props.instance);

    setLoading(true)

    setLoading(false)

  }, [props.instance]);

  const renderData = () => {

    if (loading) {
      return <ui5.BusyIndicator
        active
        delay={1000}
        size="Medium"
      />
    }

    if (error) {
      return <ui5.IllustratedMessage name="UnableToLoad" />
    }

    return (
      <ui5.Dialog
        style={{ width: "50%" }}
        ref={dialogRef}
        header={
          <ui5.Bar
            design="Header"
            startContent={
              <>
                <ui5.Title level="H5">
                  Create {instance?.name} Service Instance
                </ui5.Title>
              </>
            }
          />
        }
        footer={
          <ui5.Bar
            design="Footer"
            endContent={
              <>
                <ui5.Button onClick={handleClose}>Close</ui5.Button>
              </>
            }
          />
        }
      >
        <ui5.Panel headerLevel="H2" headerText="Service Instance Details">
          <ui5.Form>
            <ui5.FormItem label="Name">
              <ui5.Text>{instance?.name}</ui5.Text>
            </ui5.FormItem>
          </ui5.Form>
        </ui5.Panel>

        <ui5.Panel headerLevel="H2" headerText="Service Bindings">
          <ServiceBindingForm serviceInstanceID={instance?.id}/>
        </ui5.Panel>
      </ui5.Dialog>
    )
  }
  // @ts-ignore
  return <>{renderData()}</>;
})

export default ServiceInstancesDetailsView;
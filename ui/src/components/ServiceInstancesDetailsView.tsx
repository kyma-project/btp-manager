import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  ServiceInstance,
} from "../shared/models";
import { useEffect, useRef, useState } from "react";
import ServiceBindingForm from "./ServiceBindingForm";

function ServiceInstancesDetailsView(props: any) {
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(true);
  const [error] = useState(null);
  const [instance, setInstance] = useState<ServiceInstance>();
  const dialogRef = useRef(null);

  const handleClose = () => {
    setOpen(false)
  };

  useEffect(() => {
    if (!Ok(props.instance)) {
      return;
    }

    setInstance(props.instance);

    setLoading(true)

    // additional logic
    setOpen(props.open)

    setLoading(false)

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const renderData = () => {

    if (loading) {
      return <ui5.Loader progress="100%" />
    }

    if (error) {
      return <ui5.IllustratedMessage name="UnableToLoad" />
    }

    return (
      <ui5.Dialog
        open={open}
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
                <ui5.Button>Create</ui5.Button>
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
}

export default ServiceInstancesDetailsView;
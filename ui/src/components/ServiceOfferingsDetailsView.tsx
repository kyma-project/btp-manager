import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  ServiceOffering,
  ServiceOfferingDetails,
  ServiceOfferingPlan,
} from "../shared/models";
import { useEffect, useRef, useState } from "react";
import axios from "axios";
import api from "../shared/api";

function ServiceOfferingsDetailsView(props: any) {
  const [plan, setPlan] = useState<ServiceOfferingPlan>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [offering, setOffering] = useState<ServiceOffering>();
  const [details, setDetails] = useState<ServiceOfferingDetails>();
  const dialogRef = useRef(null);

  const handleClose = () => {
    // @ts-ignore
    dialogRef.current.close();
  };

  const onChangeSelect = (e: any) => {
    // @ts-ignore
    for (let i = 0; i < details?.plans.length; i++) {
      if (details?.plans[i].name === e.detail.selectedOption.dataset.id) {
        setPlan(details?.plans[i]);
      }
    }
  };

  useEffect(() => {
    if (!Ok(props.offering)) {
      return;
    }

    setLoading(true);
    axios
      .get<ServiceOfferingDetails>(api(`service-offering/${props.offering.id}`))
      .then((response) => {
        setLoading(false);
        setDetails(response.data);
        setOffering(props.offering);
        // @ts-ignore
        dialogRef.current.show();
      })
      .catch((error) => {
        setLoading(false);
        setError(error);
      });
    setLoading(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const renderData = () => {
    if (loading) {
      return <ui5.Loader progress="100%"/>
    }

    if (error) {
        return <ui5.IllustratedMessage name="UnableToLoad"/>
    }

    return (
      <>
        <ui5.Dialog
          style={{ width: "50%" }}
          ref={dialogRef}
          header={
            <ui5.Bar
              design="Header"
              startContent={
                <>
                  <ui5.Title level="H5">
                    Create {offering?.catalogName} Service Instance
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
          <ui5.Panel headerLevel="H2" headerText="Service Details">
            <ui5.Form>
              <ui5.FormItem label="Name">
                <ui5.Text>{offering?.catalogName}</ui5.Text>
              </ui5.FormItem>
              <ui5.FormItem label="Description">
                <ui5.Text>{offering?.description}</ui5.Text>
              </ui5.FormItem>
              {Ok(details?.longDescription) && (
                <ui5.FormItem label="Long Description">
                  <ui5.Text>{details?.longDescription}</ui5.Text>
                </ui5.FormItem>
              )}
              {Ok(offering?.metadata.supportUrl) && (
                <ui5.FormItem label="Support URL">
                  <ui5.Link
                    target="_blank"
                    href={offering?.metadata.supportUrl}
                  >
                    Link
                  </ui5.Link>
                </ui5.FormItem>
              )}
              {Ok(offering?.metadata.documentationUrl) && (
                <ui5.FormItem label="Documentation URL">
                  <ui5.Link
                    target="_blank"
                    href={offering?.metadata.documentationUrl}
                  >
                    Link
                  </ui5.Link>
                </ui5.FormItem>
              )}
            </ui5.Form>
          </ui5.Panel>
          <ui5.Panel headerLevel="H2" headerText="Plan Details">
            <ui5.Form>
              <ui5.FormItem label="Plan Name">
                <ui5.Select id="selectOption" onChange={onChangeSelect}>
                  {details?.plans.map(
                    (value: ServiceOfferingPlan, index: number) => {
                      if (!Ok(plan) && index == 0) {
                        setPlan(details?.plans[0]);
                      }
                      return (
                        <ui5.Option key={index} data-id={value.name} >
                          {value.name}
                        </ui5.Option>
                      );
                    }
                  )}
                </ui5.Select>
              </ui5.FormItem>
              <ui5.FormItem label="Description">
                <ui5.Text>{plan?.description}</ui5.Text>
              </ui5.FormItem>
            </ui5.Form>
          </ui5.Panel>
          <ui5.Panel accessibleRole="Form" headerLevel="H2" headerText="Create">
            <ui5.Form>
              <ui5.FormItem label="Name">
                <ui5.Input
                  style={{ width: "100vw" }}
                  required
                  value={generateServiceInstanceName(
                    plan?.name,
                    offering?.catalogName
                  )}
                ></ui5.Input>
              </ui5.FormItem>
              <ui5.FormItem label="Provisioning Parameters">
                <ui5.TextArea
                  style={{ width: "100%", height: "100px" }}
                  valueState="None"
                  title="Provisioning Parameters"
                />
              </ui5.FormItem>
              <ui5.FormItem label="External Name">
                <ui5.Input></ui5.Input>
              </ui5.FormItem>
            </ui5.Form>
          </ui5.Panel>
        </ui5.Dialog>
      </>
    )}
  // @ts-ignore
  return <>{renderData()}</>;
}

function generateServiceInstanceName(
  plan: string | undefined,
  service: string | undefined
): string {
  const id = window.crypto.randomUUID().substring(0, 4);
  return `${service}-${plan}-${id}`;
}

export default ServiceOfferingsDetailsView;
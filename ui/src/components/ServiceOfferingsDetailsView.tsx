import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
  Secret,
  ServiceOffering,
  ServiceOfferingDetails,
  ServiceOfferingPlan,
} from "../shared/models";
import { useEffect, useState } from "react";
import axios from "axios";
import api from "../shared/api";
import CreateInstanceForm from "./CreateInstanceForm";

function ServiceOfferingsDetailsView(props: any) {
  const [plan, setPlan] = useState<ServiceOfferingPlan>();
  const [secret, setSecret] = useState<Secret>(new Secret());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [offering, setOffering] = useState<ServiceOffering>();
  const [details, setDetails] = useState<ServiceOfferingDetails>();

  const onChangeSelect = (e: any) => {
    // @ts-ignore
    for (let i = 0; i < details?.plans.length; i++) {
      if (details?.plans[i].name === e.detail.selectedOption.dataset.id) {
        setPlan(details?.plans[i]);
      }
    }
  };

  useEffect(() => {
    setLoading(true);
    if (!Ok(props.offering)) {
      return;
    }

    if (!Ok(props.secret) || !Ok(props.secret.name) || !Ok(props.secret.namespace)) {
      return;
    }

    setSecret(props.secret);
    axios
      .get<ServiceOfferingDetails>(api(`service-offerings/${props.offering.id}`),
      {
        params:
        {
          sm_secret_name: props.secret.name,
          sm_secret_namespace: props.secret.namespace
        }
      })
      .then((response) => {
        setLoading(false);
        setDetails(response.data);
        setPlan(response.data?.plans[0])
        setOffering(props.offering);
        setError(null)
      })
      .catch((error) => {
        setLoading(false);
        setError(error);
      });
  }, [props.offering, props.secret]);

  const renderData = () => {
    if (loading) {
      return <ui5.BusyIndicator
      active
      delay={1}
      size="Medium"
      />
    }

    if (error) {
        return <ui5.IllustratedMessage name="UnableToLoad"/>
    }

    return (
        <div slot="midColumn">
          <ui5.Panel  headerLevel="H2" headerText="Service Instance Details">
            <ui5.Form>
              <ui5.FormItem label="Name">
                <ui5.Text>{offering?.metadata.displayName}</ui5.Text>
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
                      if (!Ok(plan) && index === 0) {
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
            <CreateInstanceForm secret={secret} plan={plan} offering={props.offering} />
          </ui5.Panel>
      </div>
    )}
  
  return <>{renderData()}</>;
}

export default ServiceOfferingsDetailsView;
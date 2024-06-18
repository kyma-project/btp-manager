import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {ServiceOffering, ServiceOfferingDetails, ServiceOfferingPlan} from "../shared/models";
import {useEffect, useRef, useState} from "react";
import axios from "axios";
import api from "../shared/api";

function ServiceOfferingsDetailsView(props: any) {
    const [plan, setPlan] = useState<ServiceOfferingPlan>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [selectedOffering, setSelectedOffering] = useState<ServiceOffering>();
    const [details, setDetails] = useState<ServiceOfferingDetails>();
    const dialogRef = useRef(null);
    const handleOpen = (offering: ServiceOffering) => {
    
    };

    const handleClose = () => {
        // @ts-ignore
        dialogRef.current.close();
    };

    const onChangeSelect = (e: any) => {
        // @ts-ignore
        for (let i = 0; i < details?.plans.length; i++) {
            if (props.details?.plans[i].name === e.detail.selectedOption.dataset.id) {
                setPlan(props.details?.plans[i]);
            }
        }
    }

    useEffect(() => {
        setLoading(true);
        axios
            .get<ServiceOfferingDetails>(api(`service-offering/${props.offering.id}`))
            .then((response) => {
                setLoading(false);
                setDetails(response.data);
                setSelectedOffering(props.offering);
                // @ts-ignore
                dialogRef.current.show();
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
            });
        setLoading(false);
    }, []);
    
    // @ts-ignore
    return <>
    <ui5.Dialog
        style={{width: "50%"}}
        ref={dialogRef}
        header={
            <ui5.Bar
                design="Header"
                startContent={
                    <>
                        <ui5.Title level="H5">Create {props.selectedOffering?.catalogName} Service
                            Instance
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
        <ui5.Panel
            headerLevel="H2"
            headerText="Service Details"
        >
            <ui5.Form>
                <ui5.FormItem label="Catalog Name">
                    <ui5.Text>{props.offering.catalogName}</ui5.Text>
                </ui5.FormItem>
                <ui5.FormItem label="Description">
                    <ui5.Text>{props.offering.description}</ui5.Text>
                </ui5.FormItem>
                {Ok(props.details?.longDescription) && <ui5.FormItem label="Long Description">
                    <ui5.Text>{props.details?.longDescription}</ui5.Text>
                </ui5.FormItem>}
                {Ok(props.offering.metadata.supportUrl) && <ui5.FormItem label="Support URL">
                    <ui5.Link target="_blank" href={props.offering.metadata.supportUrl}>Link
                    </ui5.Link>
                </ui5.FormItem>}
                {Ok(props.offering.metadata.documentationUrl) &&
                    <ui5.FormItem label="Documentation URL">
                        <ui5.Link target="_blank"
                                  href={props.offering.metadata.documentationUrl}>Link
                        </ui5.Link>
                    </ui5.FormItem>}
            </ui5.Form>
        </ui5.Panel>

        <ui5.Panel
            headerLevel="H2"
            headerText="Plan Details"
        >
            <ui5.Form>
                <ui5.FormItem label="Plan Name">
                    <ui5.Select onChange={onChangeSelect}>
                        {
                            props.details?.plans.map((plan :ServiceOfferingPlan, index :number) =>
                                (
                                    <>
                                        <ui5.Option
                                            data-id={plan.name}
                                            key={plan.name}>{plan.name}
                                        </ui5.Option>
                                    </>
                                ))
                        }
                    </ui5.Select>
                </ui5.FormItem>
                <ui5.FormItem label="Description">
                    <ui5.Text>{plan?.description}</ui5.Text>
                </ui5.FormItem>
            </ui5.Form>
        </ui5.Panel>

        <ui5.Panel
            accessibleRole="Form"
            headerLevel="H2"
            headerText="Service Instance Details"
        >
            <ui5.Form>
                <ui5.FormItem label="Name">
                    <ui5.Input style={{width: "100vw"}} required
                               value={generateServiceInstanceName(plan?.name, props.offering.catalogName)}></ui5.Input>
                </ui5.FormItem>
                <ui5.FormItem label="Provisioning Parameters">
                    <ui5.TextArea
                        style={{width: "100%", height: "100px"}}
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
}


function generateServiceInstanceName(plan: string | undefined, service: string | undefined): string {
    const id = window.crypto.randomUUID().substring(0, 4)
    return `${service}-${plan}-${id}`;
}

export default ServiceOfferingsDetailsView;
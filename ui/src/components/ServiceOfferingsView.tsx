import * as ui5 from "@ui5/webcomponents-react";
import {useEffect, useRef, useState} from "react";
import axios from "axios";
import {ServiceOffering, ServiceOfferingDetails, ServiceOfferingPlan, ServiceOfferings} from "../shared/models";
import api from "../shared/api";
import "@ui5/webcomponents-icons/dist/AllIcons.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoEntries.js"
import "@ui5/webcomponents-fiori/dist/illustrations/AllIllustrations.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoData.js";
import Ok from "../shared/validator";
import { createPortal } from "react-dom";

function ServiceOfferingsView(props: any) {
    const [offerings, setOfferings] = useState<ServiceOfferings>();
    const [selectedOffering, setSelectedOffering] = useState<ServiceOffering>();
    const [details, setDetails] = useState<ServiceOfferingDetails>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [plan, setPlan] = useState<ServiceOfferingPlan>();
    const [dialogIsOpen, setDialogIsOpen] = useState(false);
    const dialogRef = useRef(null);
    const handleOpen = (offering: ServiceOffering) => {
        // @ts-ignore
        load(offering);
        setDialogIsOpen(true);
        // @ts-ignore
        dialogRef.current.show();
    };
    const handleClose = () => {
        // @ts-ignore
        setDialogIsOpen(false);
        // @ts-ignore
        dialogRef.current.close();
    };


    const onChangeSelect = (e: any) => {
        console.log(e);
        const x = e.detail.selectedOption.dataset.id;
        // @ts-ignore
        for (let i = 0; i < details?.plans.length; i++) {
            if (details?.plans[i].name === x) {
                setPlan(details?.plans[i]);
            }
        }
    }
    
    useEffect(() => {
        if (!Ok(props.secret)) {
            return;
        }
        const secretText = splitSecret(props.secret);
        if (Ok(secretText)) {
            setLoading(true);
            axios
                .get<ServiceOfferings>(
                    api(
                        `service-offerings/${secretText.namespace}/${secretText.secretName}`
                    )
                )
                .then((response) => {
                    setLoading(false);
                    setOfferings(response.data);
                })
                .catch((error) => {
                    setLoading(false);
                    setError(error);
                });
            setLoading(false);
        }
    }, [props.secret]);

    function getImg(b64: string) {
        if (!Ok(b64) || b64 === "not found") {
            // grey color
            return "data:image/gif;base64,R0lGODlhAQABAIAAAMLCwgAAACH5BAAAAAAALAAAAAABAAEAAAICRAEAOw==";
        } else {
            return b64;
        }
    }

    function load(offering: ServiceOffering) {
        setLoading(true);
        axios
            .get<ServiceOfferingDetails>(api(`service-offering/${offering.id}`))
            .then((response) => {
                setLoading(false);
                setDetails(response.data);
                setSelectedOffering(offering);
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
            });
        setLoading(false);
    }

    const renderData = () => {
        if (loading) {
            return <ui5.Loader progress="100%"/>
        }

        if (error) {
            return <ui5.IllustratedMessage name="UnableToLoad"/>
        }

        // @ts-ignore
        if (!Ok(offerings) || !Ok(offerings.items)) {
            return <ui5.IllustratedMessage name="NoEntries"/>
        }
        return offerings?.items.map((offering, index) => {
            // @ts-ignore
            // @ts-ignore
            // @ts-ignore
            // @ts-ignore
            return (
                <>
                    <ui5.Card
                        key={index}
                        style={{
                            width: "20%",
                            height: "5%",
                        }}
                        onClick={() => {
                            handleOpen(offering);
                        }}
                        header={
                            <ui5.CardHeader
                                avatar={
                                    <ui5.Avatar>
                                        <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                                    </ui5.Avatar>
                                }
                                style={{
                                    width: "100%",
                                    height: "500px",
                                }}
                                subtitleText={offering.description}
                                titleText={offering.catalogName}
                                status={formatStatus(index, offerings?.numItems)}
                                interactive
                            />
                        }
                    >
                    </ui5.Card>

                    {createPortal(
                    <ui5.Dialog
                        style={{width: "50%"}}
                        ref={dialogRef}
                        header={
                            <ui5.Bar
                                design="Header"
                                startContent={
                                    <>
                                        <ui5.Title level="H5">Create {selectedOffering?.catalogName} Service Instance</ui5.Title>
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
                                    <ui5.Text>{selectedOffering?.catalogName}</ui5.Text>
                                </ui5.FormItem>
                                <ui5.FormItem label="Description">
                                    <ui5.Text>{selectedOffering?.description}</ui5.Text>
                                </ui5.FormItem>
                                <ui5.FormItem label="Long Description">
                                    <ui5.Text>{details?.longDescription}</ui5.Text>
                                </ui5.FormItem>
                                <ui5.FormItem label="Support URL">
                                    <ui5.Text>{selectedOffering?.metadata.supportUrl}</ui5.Text>
                                </ui5.FormItem>
                                <ui5.FormItem label="Documentation URL">
                                    <ui5.Text>{selectedOffering?.metadata.documentationUrl}</ui5.Text>
                                </ui5.FormItem>
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
                                            details?.plans.map((plan, index) =>
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
                                <ui5.FormItem label="Support URL">
                                    <ui5.Text>{plan?.supportUrl}</ui5.Text>
                                </ui5.FormItem>
                                <ui5.FormItem label="Documentation URL">
                                    <ui5.Text>{plan?.documentationUrl}</ui5.Text>
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
                                    <ui5.Input style={{width: "100vw"}} required value={generateServiceInstanceName(plan?.name, selectedOffering?.catalogName)}></ui5.Input>
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
                    </ui5.Dialog>, document.body)}
                </>
            );
        });
    };

    return <>{renderData()}</>;
}

function generateServiceInstanceName(plan :string | undefined, service :string | undefined) : string {
    const id = window.crypto.randomUUID().substring(0,4)
    return `${service}-${plan}-${id}`;
}

function splitSecret(secret: string) {
    if (secret == null) {
        return {};
    }
    const secretParts = secret.split(" ");
    const secretName = secretParts[0];
    let namespace = secretParts[2].replace("(", "");
    namespace = namespace.replace(")", "");
    return {secretName, namespace};
}

function formatStatus(i: number, j: number) {
    return `${++i} of ${j}`;
}

export default ServiceOfferingsView;

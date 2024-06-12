import * as ui5 from "@ui5/webcomponents-react";
import {useEffect, useRef, useState} from "react";
import {createPortal} from "react-dom";
import axios from "axios";
import {ServiceOfferingDetails, ServiceOfferings} from "../shared/models";
import api from "../shared/api";
import "@ui5/webcomponents-icons/dist/AllIcons.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoEntries.js"
import "@ui5/webcomponents-fiori/dist/illustrations/AllIllustrations.js"
import Ok from "../shared/validator";

function ServiceOfferingsView(props: any) {
    const [offerings, setOfferings] = useState<ServiceOfferings>();
    const [serviceOfferingDetails, setServiceOfferingDetails] = useState<ServiceOfferingDetails>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const dialogRef = useRef(null);
    
    const handleOpen = (id: any) => {
        // @ts-ignore
        dialogRef.current.show();
        load(id);
    };
    const handleClose = () => {
        // @ts-ignore
        dialogRef.current.close();
    };
    
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


    if (loading) {
        return <ui5.Loader progress="100%"/>
    }

    if (error) {
        return <ui5.IllustratedMessage name="UnableToLoad" />
    }

    function getImg(b64: string) {
        if (!Ok(b64) || b64 === "not found") {
            // grey color
            return "data:image/gif;base64,R0lGODlhAQABAIAAAMLCwgAAACH5BAAAAAAALAAAAAABAAEAAAICRAEAOw==";
        } else {
            return b64;
        }
    }

    function load(id: string) {
        setLoading(true);
        axios
            .get<ServiceOfferingDetails>(api(`service-offering/${id}`))
            .then((response) => {
                setLoading(false);
                setServiceOfferingDetails(response.data);
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
            });
        setLoading(false);
    }
    
    const renderData = () => {
        // @ts-ignore
        if (!Ok(offerings) || !Ok(offerings.items)) {
            return <ui5.IllustratedMessage name="NoEntries" />
        }
        return offerings?.items.map((offering, index) => {
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
                            handleOpen(offering.id);
                        }}
                        header={
                            <ui5.CardHeader
                                avatar={
                                    <ui5.Avatar>
                                        <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                                    </ui5.Avatar>
                                }
                                subtitleText={offering.description}
                                titleText={offering.catalogName}
                                status={formatStatus(index, offerings?.numItems)}
                                interactive
                            />
                        }
                    >
                    </ui5.Card>

                    <>
                        {createPortal(
                            <ui5.Dialog
                                style={{width: "50%"}}
                                ref={dialogRef}
                                className="headerPartNoPadding footerPartNoPadding"
                                footer={
                                    <ui5.Bar
                                        design="Footer"
                                        endContent={
                                            <ui5.Button onClick={handleClose}>Close</ui5.Button>
                                        }
                                    />
                                }
                            >
                                    <ui5.Panel
                                        headerLevel="H2"
                                        headerText="Service Offering Details"
                                    >
                                        <ui5.Text>{serviceOfferingDetails?.longDescription}</ui5.Text>
                                    </ui5.Panel>
                                
                                    <ui5.Panel
                                        headerLevel="H2"
                                        headerText="Plan Details"
                                    >
                                        <ui5.Form>
                                            <ui5.FormItem label="Plan Name">
                                                <ui5.Select>
                                                    {
                                                        serviceOfferingDetails?.plans.map((plan, index) =>
                                                            (
                                                                <ui5.Option
                                                                    key={index}>{plan.name}
                                                                </ui5.Option>
                                                            ))
                                                    }
                                                </ui5.Select>
                                            </ui5.FormItem>
                                        </ui5.Form>
                                    </ui5.Panel>

                                <ui5.Panel
                                    accessibleRole="Form"
                                    headerLevel="H2"
                                    headerText="Create Service Instance"
                                >
                                    <ui5.Form>
                                        <ui5.FormItem label="Name">
                                            <ui5.Input></ui5.Input>
                                        </ui5.FormItem>
                                        <ui5.FormItem label="External Name">
                                        <ui5.Input></ui5.Input>
                                        </ui5.FormItem>
                                        <ui5.FormItem label="Provisioning Parameters">
                                            <ui5.TextArea
                                                style={{width: "100%"}}
                                                valueState="None"
                                                title="Provisioning Parameters"
                                            />
                                        </ui5.FormItem>
                                    </ui5.Form>
                                </ui5.Panel>
                            </ui5.Dialog>,
                            document.body
                        )}
                    </>
                </>
            );
        });
    };

    return <>{renderData()}</>;
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

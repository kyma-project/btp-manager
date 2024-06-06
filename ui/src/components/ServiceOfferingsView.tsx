import * as ui5 from "@ui5/webcomponents-react";
import {useEffect, useRef, useState} from "react";
import {createPortal} from "react-dom";
import axios from "axios";
import {ServiceOfferingDetails, ServiceOfferings} from "../shared/models";
import api from "../shared/api";
import "@ui5/webcomponents-icons/dist/AllIcons.js"
import Ok from "../shared/validator";

function ServiceOfferingsView(props: any) {
    const [offerings, setOfferings] = useState<ServiceOfferings>();
    const [offeringsCachced, setOfferingsCached] = useState<ServiceOfferings>();
    const [serviceOfferingDetails, setServiceOfferingDetails] =
        useState<ServiceOfferingDetails>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [labels, setLabels] = useState<JSX.Element>();
    const dialogRef = useRef(null);
    const [planDesc, setPlanDesc] = useState<string>();
    const handleOpen = (id: any) => {
        // @ts-ignore
        dialogRef.current.show();
        console.log("handleOpen id: ", id)
        load(id);
    };
    const handleClose = () => {
        // @ts-ignore
        dialogRef.current.close();
    };


    useEffect(() => {
        if (props.secret == null) {
            return;
        }
        const splited = splitSecret(props.secret);
        if (splited) {
            setLoading(true);
            axios
                .get<ServiceOfferings>(
                    api(
                        `service-offerings/${splited.namespace}/${splited.secretName}`
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
        }
    }, []);


    if (loading) {
        return <ui5.Loader progress="60%"/>
    }

    if (error) {
        return <ui5.IllustratedMessage name="UnableToLoad" style={{height: "50vh", width: "30vw"}}/>
    }

    function filter(e: string) {
        return;
        console.log("filtering")
        // @ts-ignore
        setOfferings(offeringsCachced);
        const ServiceOfferings = offeringsCachced;
        let ii = 0;
        // @ts-ignore
        for (let i = 0; i < offerings?.items.length; i++) {
            // @ts-ignore
            if (offerings?.items[i].metadata.displayName.includes(e)) {
                // @ts-ignore
                ServiceOfferings.items[ii] = [offerings?.items[i]];
                ii++;
            }
        }
    }

    function getImg(b64: string) {
        if (b64 == null) {
            return "";
        } else {
            return b64;
        }
    }

    function load(id: string) {
        console.log("loading id: ", id)
        setLoading(true);
        axios
            .get<ServiceOfferingDetails>(api(`service-offering-details/${id}`))
            .then((response) => {
                console.log("load response: ", response)
                setLoading(false);
                setServiceOfferingDetails(response.data);
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
                console.log("load error: ", error)
            });
        setLoading(false);
    }

    const renderData = () => {
        // @ts-ignore
        if (!Ok(offerings) || !Ok(offerings.items)) {
            return <ui5.IllustratedMessage name="NoEntries" style={{height: "50vh", width: "30vw"}}/>
        }
        //console.log("renderData")
        //filter(props.phrase)

        return offerings?.items.map((offering, index) => {
            // @ts-ignore
            // @ts-ignore
            return (
                <>
                    <ui5.Card
                        key={index}
                        style={{
                            width: "20%",
                            height: "0",
                            padding: "10px"
                        }}
                        onClick={() => {
                            console.log("opening id: ", offering.id)
                            handleOpen(offering.id);
                        }}
                        header={
                            <ui5.CardHeader
                                avatar={
                                    <ui5.Avatar>
                                        <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                                    </ui5.Avatar>
                                }
                                subtitleText={offering.metadata.displayName}
                                titleText={offering.catalogName}
                                status={formatStatus(index, offerings?.numItems)}
                                interactive
                            />
                        }
                    ></ui5.Card>

                    <>
                        {createPortal(
                            <ui5.Dialog
                                style={{width: "800px"}}
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
                                onAfterClose={function _a() {
                                }}
                                onAfterOpen={function _a() {
                                }}
                                onBeforeClose={function _a() {
                                }}
                                onBeforeOpen={function _a() {
                                }}
                            >
                                <div>
                                    <ui5.Panel
                                        headerLevel="H2"
                                        headerText="Service Offering Details"
                                    >
                                        <ui5.Text>{serviceOfferingDetails?.longDescription}</ui5.Text>
                                    </ui5.Panel>
                                </div>

                                <ui5.Panel
                                    accessibleRole="Form"
                                    headerLevel="H2"
                                    headerText="Create Service Instance"
                                    onToggle={function _a() {
                                    }}
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
                                                onChange={function _a() {
                                                }}
                                                onInput={function _a() {
                                                }}
                                                onScroll={function _a() {
                                                }}
                                                onSelect={function _a() {
                                                }}
                                                valueState="None"
                                                title="Provisioning Parameters"
                                            />
                                        </ui5.FormItem>
                                        <ui5.FormItem label="Plan Name">
                                            <ui5.Select
                                            >
                                                {
                                                    serviceOfferingDetails?.plans.map((plan, index) =>
                                                        (
                                                            <ui5.Option onLoad={() => {
                                                                console.log("onLoad")
                                                                setPlanDesc(plan.description);
                                                            }} onClick={() => {
                                                                console.log("onClick")
                                                            }} onFocus={() => {
                                                                console.log("onFocus")
                                                            }}
                                                                        key={index}>{plan.name} X
                                                            </ui5.Option>
                                                        ))}
                                            </ui5.Select>
                                            <ui5.Text>{planDesc}</ui5.Text>
                                        </ui5.FormItem>
                                        <ui5.Icon name="add"
                                                  onClick={() => {
                                                      const it = (
                                                          <ui5.FormItem>
                                                              <ui5.Input style={{width: "50px"}}></ui5.Input>
                                                              <ui5.Input style={{width: "50px"}}></ui5.Input>
                                                          </ui5.FormItem>
                                                      );
                                                      // @ts-ignore
                                                      setLabels([...labels, it]);
                                                  }}
                                        />
                                        <ui5.FormItem label="Labels">{labels}</ui5.FormItem>
                                        <ui5.FormItem label="Annotations">
                                            <ui5.Icon name="add"/>
                                            <ui5.FormItem>
                                                <ui5.Input style={{width: "50px"}}></ui5.Input>
                                            </ui5.FormItem>
                                            <ui5.FormItem>
                                                <ui5.Input style={{width: "50px"}}></ui5.Input>
                                            </ui5.FormItem>
                                        </ui5.FormItem>
                                        <ui5.FormItem>
                                            <ui5.Button
                                                style={{width: "100px"}}
                                                onClick={function _a() {
                                                }}
                                            >
                                                Create
                                            </ui5.Button>
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

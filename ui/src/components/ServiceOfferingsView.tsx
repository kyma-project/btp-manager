import * as ui5 from "@ui5/webcomponents-react";
import {useEffect, useState} from "react";
import axios from "axios";
import {ServiceOfferings} from "../shared/models";
import api from "../shared/api";
import "@ui5/webcomponents-icons/dist/AllIcons.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoEntries.js"
import "@ui5/webcomponents-fiori/dist/illustrations/AllIllustrations.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoData.js";
import Ok from "../shared/validator";
import {createPortal} from "react-dom";
import ServiceOfferingsDetailsView from "./ServiceOfferingsDetailsView";

function ServiceOfferingsView(props: any) {
    const greyImg = "data:image/gif;base64,R0lGODlhAQABAIAAAMLCwgAAACH5BAAAAAAALAAAAAABAAEAAAICRAEAOw=="
    const [offerings, setOfferings] = useState<ServiceOfferings>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [portal, setPortal] = useState<JSX.Element>();
    
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
            return greyImg;
        }
        return b64;
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
        const cards = offerings?.items.map((offering, index) => {
            // @ts-ignore
            return (
                <>
                    <ui5.Card
                        key={index}
                        style={{
                            width: '600px',
                        }}
                        onClick={() => {
                            setPortal(createPortal( <ServiceOfferingsDetailsView offering={offering} />, document.getElementById("App")!!, window.crypto.randomUUID()))
                        }}
                        header={
                            <ui5.CardHeader
                                avatar={
                                    <ui5.Avatar>
                                        <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                                    </ui5.Avatar>
                                }
                                style={{
                                    width: "100%"
                                }}
                                subtitleText={offering.description}
                                titleText={offering.catalogName}
                                status={formatStatus(index, offerings?.numItems)}
                                interactive
                            />
                        }
                    />
                </>
            );
        });

        return <>
            {cards}
            {portal != null && portal}
        </>
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

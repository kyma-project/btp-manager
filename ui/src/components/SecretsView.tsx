import * as ui5 from "@ui5/webcomponents-react";
import axios from "axios";
import {useEffect, useState} from "react";
import {Secrets, Secret} from "../shared/models";
import Ok from "../shared/validator";
import api from "../shared/api";
import ServiceOfferingsView from "./ServiceOfferingsView";

function SecretsView(props: any) {
    const [secrets, setSecrets] = useState<Secrets>();
    const [selectedSecret, setSelectedSecret] = useState<Secret>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<Secrets>(api("secrets"))
            .then((response) => {
                setLoading(false);
                setSecrets(response.data);
                if (Ok(response.data) && Ok(response.data.items)) {
                    setSelectedSecret(response.data.items[0])
                    const secret = formatSecretText(response.data.items[0].name, response.data.items[0].namespace)
                    props.handler(secret);
                    props.setPageContent(<ServiceOfferingsView secret={secret}/>);
                } else {
                    props.handler(formatSecretText("", ""));
                }
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
                setSecrets(undefined);
                props.handler(formatSecretText("", ""));
            });
        setLoading(false);
    }, [props]);

    if (loading) {
        return <ui5.IllustratedMessage name="UnableToLoad"/>
    }

    if (error) {
        props.handler(formatSecretText("", ""));
        return <ui5.IllustratedMessage name="UnableToLoad"/>
    }

    const renderData = () => {
        // @ts-ignore
        if (!Ok(secrets) || !Ok(secrets.items)) {
            return <div>
                <>
                    <ui5.Option key={0}>{formatSecretText("", "")}</ui5.Option>
                </>
            </div>
        }
        return secrets?.items.map((secret, index) => {
            return (
                <ui5.Option key={index}>{formatSecretText(secret.name, secret.namespace)}</ui5.Option>
            );
        });
    };

    return (
        <div>
            <>
                <div>
                    <ui5.Select
                        style={{width: "20vw"}}
                        onChange={(e) => {
                            // @ts-ignore
                            const secret = e.target.value;
                            props.handler(secret);
                            props.setPageContent(<ServiceOfferingsView secret={secret}/>);
                        }}
                    >
                        {renderData()}
                    </ui5.Select>
                </div>
            </>
        </div>
    );
}

function formatSecretText(secretName: string, secretNamespace: string) {
    if (secretName === "" || secretNamespace === "") {
        return "No secret found"
    }
    return `${secretName} in (${secretNamespace})`;
}

export default SecretsView;

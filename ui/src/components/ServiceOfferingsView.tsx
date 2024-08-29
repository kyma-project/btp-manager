import * as ui5 from "@ui5/webcomponents-react";
import { useEffect, useState } from "react";
import axios from "axios";
import { Secret, ServiceOffering, ServiceOfferings } from "../shared/models";
import api from "../shared/api";
import "@ui5/webcomponents-icons/dist/AllIcons.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoEntries.js"
import "@ui5/webcomponents-fiori/dist/illustrations/AllIllustrations.js"
import "@ui5/webcomponents-fiori/dist/illustrations/NoData.js";
import Ok from "../shared/validator";
import ServiceOfferingsDetailsView from "./ServiceOfferingsDetailsView";
import { FCLLayout, FlexibleColumnLayout, ResponsiveGridLayout } from "@ui5/webcomponents-react";
import { splitSecret } from "../shared/common";
import StatusMessage from "./StatusMessage";

function ServiceOfferingsView(props: any) {
    const greyImg = "data:image/svg+xml;base64,PHN2ZyBpZD0icGxhY2Vob2xkZXIiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzVhN2E5NDt9LmNscy0ye2ZpbGw6IzA0OTFkMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPnBsYWNlaG9sZGVyPC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00Ni45NTMsMjAuNTg4YTQuMzYzLDQuMzYzLDAsMCwwLTEuODM3LS40NTksMy4yOTEsMy4yOTEsMCwwLDAtMy40LDMuMzc2LDQuMDg0LDQuMDg0LDAsMCwwLC45LDIuNjI1LDMuMDExLDMuMDExLDAsMCwwLDIuNSwxLjEyNiwzLjA4NSwzLjA4NSwwLDAsMCwxLjQ2Mi0uMzc1LDcuNTEyLDcuNTEyLDAsMCwwLDEuMzItLjg5MSwxMC4xMzUsMTAuMTM1LDAsMCwxLDEuMjI2LS44OTEsMi4yNywyLjI3LDAsMCwxLDEuMTc5LS4zNzVBMS41LDEuNSwwLDAsMSw1MiwyNi40MTJWMzkuMDcxYTIuODQzLDIuODQzLDAsMCwxLS41NzYsMiwyLjkyNiwyLjkyNiwwLDAsMS0yLjE1OS42MjZxLTIuOTIzLDAtNC4zODUuMDQ3dC0yLjEyMi4wNDdINDEuOTFhMy4zMjEsMy4zMjEsMCwwLDAsLjYuNjQ0LDUuNzE3LDUuNzE3LDAsMCwxLDIuMDc0LDQuMjIsNS4wNTQsNS4wNTQsMCwwLDEtMS42NSwzLjc1MUE1LjMzMSw1LjMzMSwwLDAsMSwzOS4xMTgsNTJhNS42LDUuNiwwLDAsMS00LjA1NS0xLjU0Nyw1LjA3MSw1LjA3MSwwLDAsMS0xLjYtMy44LDQuODYyLDQuODYyLDAsMCwxLC41MTktMi4zLDExLjQwNywxMS40MDcsMCwwLDEsMS41MTYtMS45NywyLjMzMywyLjMzMywwLDAsMCwuNDc1LS42OUgyOC4zM2ExLjM5NCwxLjM5NCwwLDAsMS0xLjA4NC0uNDY5LDIuMDExLDIuMDExLDAsMCwxLS41MTktMS4wMzJWMTUuOTA5YTEuOCwxLjgsMCwwLDEsLjQyNC0xLjE3MiwxLjQ0NCwxLjQ0NCwwLDAsMSwxLjE3OS0uNTE2aDcuNzMzYTEuOTQ5LDEuOTQ5LDAsMCwwLS4zNzctLjU2MmwtLjgtMS4xNzFhOC43ODgsOC43ODgsMCwwLDEtLjg0Ny0xLjUsNC43ODMsNC43ODMsMCwwLDEtLjQwNi0xLjY3NkE1LjM0OCw1LjM0OCwwLDAsMSwzOS4wODEsNGE1LjU1Miw1LjU1MiwwLDAsMSwzLjc5LDEuNTUzQTQuNjM1LDQuNjM1LDAsMCwxLDQ0LjU1LDkuMzQ1Yy0uMDI4LDEuNjg4LTIuMDIzLDQuMTI1LTIuMjQxLDQuMzc1YTEuNTc2LDEuNTc2LDAsMCwwLS4zLjVoNy4yNjFBMi42NSwyLjY1LDAsMCwxLDUyLDE2Ljg0N3Y0LjEyNnEwLDEuNzgyLTEuNywxLjc4MmExLjc0MywxLjc0MywwLDAsMS0xLjMxOS0uNTQ5QTEzLjE1MiwxMy4xNTIsMCwwLDAsNDYuOTUzLDIwLjU4OFpNMjguMzMsMzkuMDcxYS41ODIuNTgyLDAsMCwwLC42Ni42NTdoNy4xNjdhMS41NzksMS41NzksMCwwLDEsMS43OTIsMS43ODEsMi4yMzgsMi4yMzgsMCwwLDEtLjM4NywxLjI1NGMtLjI4My40MDgtLjU4Mi44MTMtLjksMS4yMTlzLS42MTMuODMtLjksMS4yNjZhMi41NDYsMi41NDYsMCwwLDAtLjQyNCwxLjQwNywzLjExNSwzLjExNSwwLDAsMCwxLjEzMSwyLjUzMiw0LjAyMiw0LjAyMiwwLDAsMCwyLjY0MS45MzgsMy43NzYsMy43NzYsMCwwLDAsMi40NTItLjkzOEEzLjExNSwzLjExNSwwLDAsMCw0Mi43LDQ2LjY1NWEyLjU0NiwyLjU0NiwwLDAsMC0uNDI0LTEuNDA3LDEyLjUxMywxMi41MTMsMCwwLDAtLjk0My0xLjI2NnEtLjUxOS0uNjA5LS45NDMtMS4xNzJhMi4yNjEsMi4yNjEsMCwwLDEtLjQ2Mi0xLjMsMS42MTQsMS42MTQsMCwwLDEsLjU2Ni0xLjMxMywyLjAwNiwyLjAwNiwwLDAsMSwxLjMyLS40NjhoNy40NXEuOTQyLDAsLjk0My0uNjU3VjI2LjUwNmExLjYwOSwxLjYwOSwwLDAsMC0uNzA3LjQyMnEtLjUxOS40MjEtMS4xNzkuODlhMTEuMDY5LDExLjA2OSwwLDAsMS0xLjUwOS44OTEsMy43NywzLjc3LDAsMCwxLTEuNy40MjIsNS40NSw1LjQ1LDAsMCwxLTMuNjc4LTEuNSw0LjI1LDQuMjUsMCwwLDEtMS4yMjYtMS44NzYsNy4wNTMsNy4wNTMsMCwwLDEtLjM3Ny0yLjI1LDUuMTY2LDUuMTY2LDAsMCwxLDEuNi0zLjcsNS4wMDksNS4wMDksMCwwLDEsMy42NzgtMS42NDEsNC44ODQsNC44ODQsMCwwLDEsMi4zNTcuNTE1QTcuNTg3LDcuNTg3LDAsMCwxLDQ5LjUxOCwyMC4yYy41MDYuNTg4Ljc4NS42MjQuNzg1LjYyNFYxNi44NDdhLjU0NC41NDQsMCwwLDAtLjMzMS0uNDY5LDEuNDIyLDEuNDIyLDAsMCwwLS43MDctLjE4N2gtNy40NWEyLjE0NywyLjE0NywwLDAsMS0xLjMyLS40MjIsMS41ODcsMS41ODcsMCwwLDEtLjU2Ni0xLjM2LDIuMDY3LDIuMDY3LDAsMCwxLC40MjUtMS4xNzJxLjQyNS0uNjA5Ljk0My0xLjIxOWExMi4yMjIsMTIuMjIyLDAsMCwwLC45NDMtMS4yNjYsMi41NDEsMi41NDEsMCwwLDAsLjQyNC0xLjQwNywzLjExOCwzLjExOCwwLDAsMC0xLjEzMi0yLjUzMiwzLjc3MSwzLjc3MSwwLDAsMC0yLjQ1MS0uOTM4LDMuODM5LDMuODM5LDAsMCwwLTIuNTk0LjkzOEEzLjE3OCwzLjE3OCwwLDAsMCwzNS40LDkuMzQ1YTIuNzc2LDIuNzc2LDAsMCwwLC40MjQsMS40NTQsMTAuMDM3LDEwLjAzNywwLDAsMCwuOSwxLjI2NWwuODQ5LDEuMjJhMi45MDksMi45MDksMCwwLDEsLjQ3MSwxLjEyNSwxLjYyNSwxLjYyNSwwLDAsMS0uNTE4LDEuMzYsMS45NTYsMS45NTYsMCwwLDEtMS4yNzQuNDIySDI5LjA4NHEtLjc1NSwwLS43NTQuNjU2Wm0yMy42NywwYTIuNywyLjcsMCwwLDEtLjU3NiwyLDIuNjc1LDIuNjc1LDAsMCwxLTIuMTU5LjYyNiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTM3LjE0NywzMS4wNzRhMy4zMjgsMy4zMjgsMCwwLDAtMi44NzgtMS4zNiw0LjQ0NSw0LjQ0NSwwLDAsMC0yLjEyLjQyMiw2LjE4NSw2LjE4NSwwLDAsMC0xLjE3OC44OTFxLS41NjcuNDcxLTEuMTMyLjg5MWMtLjM3My4yNzgtLjgwOC43NzMtMS4zLjc3NkgyNi43MjdWMTYuNDZhMy4zMzUsMy4zMzUsMCwwLDAtLjM3Ny0xLjUsMS40MzYsMS40MzYsMCwwLDAtMS40MTUtLjc1MUgxOS4yNzdjLS41LDAtLjc1NC4yNTEtLjc1NC44NDRhMS45MDcsMS45MDcsMCwwLDAsLjM3NywxLjEyNiw5LjE0Niw5LjE0NiwwLDAsMCwuOTQzLDEuMTI1LDUuMzQxLDUuMzQxLDAsMCwxLC45NDMsMS4yNjYsMy4yMzYsMy4yMzYsMCwwLDEsLjM3NywxLjU0Nyw0LjQ1NCw0LjQ1NCwwLDAsMS0xLjI3MywzLjE0MSw0LjA0OSw0LjA0OSwwLDAsMS0zLjA2NSwxLjM2LDMuOSwzLjksMCwwLDEtMy4wMTgtMS4zNiw0LjU0Nyw0LjU0NywwLDAsMS0xLjIyNS0zLjE0MSwyLjkzNiwyLjkzNiwwLDAsMSwuNDI0LTEuNTQ3LDEzLjU0OCwxMy41NDgsMCwwLDEsLjktMS4zMTNjLjMxNC0uNDA2LjYyNy0uNzgxLjk0My0xLjEyNWExLjU4OCwxLjU4OCwwLDAsMCwuNDcxLTEuMDc5cTAtLjg0My0xLjAzNy0uODQ0SDUuN2ExLjU4NywxLjU4NywwLDAsMC0xLjIyNi41MTZBMS44MDYsMS44MDYsMCwwLDAsNCwxNS45OTFWMzkuOWExLjgsMS44LDAsMCwwLC40NzEsMS4yNjYsMS41ODMsMS41ODMsMCwwLDAsMS4yMjYuNTE2aDguNDg4Yy42OTEsMCwxLjAzNS4yMzgsMS4wMzcuNzVhMS41NDcsMS41NDcsMCwwLDEtLjQyMi45NDRMMTMuODA3LDQ0LjVhNi41NDksNi41NDksMCwwLDAtLjk5LDEuMjY2LDMuMTE2LDMuMTE2LDAsMCwwLS40MjQsMS42NDEsNC4yMzcsNC4yMzcsMCwwLDAsMS4zNjcsMy40Nyw0Ljc5MSw0Ljc5MSwwLDAsMCw2LjIyNC0uMDQ3LDQuNTE3LDQuNTE3LDAsMCwwLDEuNDQ1LTMuMjgzLDMuNjMxLDMuNjMxLDAsMCwwLS41MTQtMS44ODljLS4yMTUtLjMwNy0uOTc4LTEuMTU4LS45NzgtMS4xNThMMTguOSw0My4zNzNhMS40OTIsMS40OTIsMCwwLDEtLjM3Ny0uOTM4cTAtLjc1Ljg0OC0uNzVoNS42NThxMS4yMjYsMCwxLjctMS41VjM1LjM0MUgyOC4zNWMuNTU3LDAsMS4wNTQuNTE5LDEuNDg5LjhhMTIuMjkxLDEyLjI5MSwwLDAsMSwxLjIyNi44OTFxLjU2NS40NjksMS4xNzkuODlhMy43ODYsMy43ODYsMCwwLDAsMS44MTYuNDIyLDMuMjU2LDMuMjU2LDAsMCwwLDMuMDg3LTEuNDA2LDUuMTE5LDUuMTE5LDAsMCwwLC45OS0zQTQuNzg4LDQuNzg4LDAsMCwwLDM3LjE0NywzMS4wNzRaIi8+PC9zdmc+"
    const [offerings, setOfferings] = useState<ServiceOfferings>();
    const [selectedOffering, setSelectedOffering] = useState<ServiceOffering>();
    const [secret, setSecret] = useState<Secret>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [layout, setLayout] = useState(FCLLayout.OneColumn);

    useEffect(() => {
        setLoading(true);

        if (!Ok(props.setTitle)) {
            setSecret(new Secret());
            setLoading(false);
            return;
        }
        props.setTitle("Service Offerings");

        if (!Ok(props.secret)) {
            setSecret(new Secret());
            setLoading(false);
            return;
        }
        const secret = splitSecret(props.secret);
        if (Ok(secret)) {
            setSecret(secret);
            axios
                .get<ServiceOfferings>(
                    api(
                        `service-offerings`
                    ), {
                    params:
                    {
                        sm_secret_name: secret.name,
                        sm_secret_namespace: secret.namespace
                    }
                }
                )
                .then((response) => {
                    setLoading(false);
                    setError(null);
                    setOfferings(response.data);
                })
                .catch((error) => {
                    setLoading(false);
                    setError(error);
                });
        } else {
            setLoading(false);
        }
    }, [props, props.secret]);

    function getImg(b64: string) {
        if (!Ok(b64) || b64 === "not found") {
            return greyImg;
        }
        return b64;
    }

    const renderData = () => {
        if (loading) {
            return <ui5.BusyIndicator
                active
                delay={1}
                size="Medium"
            />
        }

        if (error) {

            return <>
                <div className="margin-wrapper">
                    <StatusMessage error={error ?? undefined} success={undefined} />
                    <ui5.IllustratedMessage name="UnableToLoad" />
                </div>
            </>
        }

        // @ts-ignore
        if (!Ok(offerings) || !Ok(offerings.items)) {
            return <ui5.IllustratedMessage name="NoEntries" />
        }
        const cards = offerings?.items.map((offering, index) => {
            // @ts-ignore
            return (
                <ui5.Card
                    selection-mode="Single"
                    key={index}
                    onClick={() => {
                        setSelectedOffering(offering)
                        setLayout(FCLLayout.TwoColumnsMidExpanded)
                    }}
                    header={
                        <ui5.CardHeader
                            avatar={
                                <ui5.Avatar>
                                    <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                                </ui5.Avatar>
                            }
                            subtitleText={offering.catalog_name}
                            titleText={offering.metadata.displayName}
                            status={formatStatus(index, offerings?.numItems)}
                            interactive
                        />
                    }
                >
                </ui5.Card>
            );
        });

        return <>

            <FlexibleColumnLayout id="fcl" layout={layout}>
                <ResponsiveGridLayout selection-mode="Single" slot="startColumn" className="margin-wrapper"
                    columnsXL={3}
                    columnsL={2}
                    columnsM={1}
                    columnsS={1}
                >
                    {cards}
                </ResponsiveGridLayout>

                <div slot="midColumn" >
                    <ui5.Bar>
                        <div className="icons-container" slot="endContent">
                            <ui5.Button design="Transparent" icon="full-screen" onClick={() => {
                                if (layout === FCLLayout.MidColumnFullScreen) {
                                    setLayout(FCLLayout.TwoColumnsMidExpanded)
                                } else {
                                    setLayout(FCLLayout.MidColumnFullScreen)
                                }
                            }}></ui5.Button>
                            <ui5.Button icon="decline" design="Transparent" onClick={() => {
                                setLayout(FCLLayout.OneColumn)
                            }}></ui5.Button>
                        </div>
                    </ui5.Bar>
                    <ServiceOfferingsDetailsView secret={secret} offering={selectedOffering} />
                </div>
            </FlexibleColumnLayout>
        </>
    };



    return <>{renderData()}</>;
}

function formatStatus(i: number, j: number) {
    return `${++i} of ${j}`;
}

export default ServiceOfferingsView;

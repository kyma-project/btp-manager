import ServiceOfferings from "./ServiceOfferingsView";
import ServiceInstancesView from "./ServiceInstancesView";
import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./SecretsView";
import React, { useEffect } from "react";

function Overview(props: any) {
    const [secret, setSecret] = React.useState(null);
    const [pageContent, setPageContent] = React.useState<JSX.Element>();

    function handler(s: any) {
        setSecret(s);
    }
    
    useEffect(() => {
        setPageContent(<ServiceOfferings secret={secret}/>)
    }, [secret]);
    
    return (
        <>
        
            <ui5.Bar
                design="Header"
                endContent={<span>SAP BTP, Kyma runtime</span>}
                startContent={<span>Select your credentials:</span>}
            >
                <Secrets handler={(e: any) => handler(e)} style={{width: "100%"}} />
            </ui5.Bar>


            <div className="flex-container flex-row">
            


            <>
                    {
                    }

                        <div className="margin-wrapper">

                            <ui5.SideNavigation>
                                <ui5.SideNavigationItem
                                    text="Marketplace"
                                    icon="puzzle"
                                    selected
                                    onClick={() => {
                                        setPageContent(<ServiceOfferings secret={secret}/>);
                                    }}
                                />
                                <ui5.SideNavigationItem
                                    text="Service Instances"
                                    icon="connected"
                                    onClick={() => {
                                        setPageContent(<ServiceInstancesView/>);
                                    }}
                                />
                            </ui5.SideNavigation>
                        </div>

                        
                        <div className="margin-wrapper scrollable">
                            {pageContent}
                        </div>
            </>
        </div>

        </>
    );
}

export default Overview;

import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./SecretsView";
import { matchPath, Outlet, useLocation } from "react-router-dom";
import { useNavigate } from "react-router-dom";


function Layout({ onSecretChanged }: { onSecretChanged: (secret: string) => void }) {
    const navigate = useNavigate();
    const location = useLocation();
    return (
        <>
            <ui5.Bar
                design="Header"
                endContent={<span>SAP BTP, Kyma runtime</span>}
                startContent={<span>Select your credentials:</span>}
            >
                <Secrets onSecretChanged={(secret: string) => onSecretChanged(secret)} />
            </ui5.Bar>


            <div className="flex-container flex-row">
                <>
                    <div className="margin-wrapper">

                        <ui5.SideNavigation>
                            <ui5.SideNavigationItem
                                text="Marketplace"
                                icon="puzzle"
                                selected={!!matchPath(
                                        location.pathname, 
                                        '/offerings'
                                      )
                                }
                                onClick={() => {
                                    navigate("/offerings");
                                }}
                            />

                            <ui5.SideNavigationItem
                                text="Service Instances"
                                icon="connected"
                                selected={!!matchPath(
                                        location.pathname, 
                                        '/instances'
                                      )
                                }
                                onClick={() => {
                                    navigate("/instances");
                                }}
                            >
                                
                            </ui5.SideNavigationItem>
                        </ui5.SideNavigation>
                    </div>

                    <div className="margin-wrapper scrollable">
                        <Outlet />
                    </div>
                </>
            </div>
        </>
    );
}

export default Layout;

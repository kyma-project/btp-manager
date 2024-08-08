import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./SecretsView";
import { matchPath, Outlet, useLocation } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { ObjectPage } from "@ui5/webcomponents-react";


function Layout({ onSecretChanged }: { onSecretChanged: (secret: string) => void }) {
    const navigate = useNavigate();
    const location = useLocation();
    return (
        <>

            <div className="margin-wrapper">

                <ui5.ShellBar style={{ "borderRadius": "var(--_ui5-v1-24-7_side_navigation_border_radius);" }}
                    logo={<img alt="SAP Logo" src="https://sap.github.io/ui5-webcomponents/images/sap-logo-svg.svg" />}
                    secondaryTitle="SAP BTP, Kyma runtime"
                    primaryTitle="BTP Manager UI"
                >

                </ui5.ShellBar>
            </div>




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

                    <div className="margin-wrapper main-column">
                    


                        <ObjectPage className="scrollable flex-column"
                              headerTitle={
                                <Secrets onSecretChanged={(secret: string) => onSecretChanged(secret)} />
                              }
                        >

                            <Outlet />
                        </ObjectPage>
                    </div>
                </>
            </div>
        </>
    );
}

export default Layout;

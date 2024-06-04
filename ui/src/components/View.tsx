import ServiceOfferings from "./ServiceOfferingsView";
import ServiceInstancesView from "./ServiceInstancesView";
import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./SecretsView";
import React from "react";

function Overview(props: any) {
  const [secret, setSecret] = React.useState(null);
  const [pageContent, setPageContent] = React.useState<JSX.Element>();
  function handler(e: any) {
    setSecret(e);
  }

  return (
    <>
      <ui5.Page
        header={<ui5.Bar design="Header">Service Management UI</ui5.Bar>}
      >
        <ui5.Title level="H1">Service Marketplace</ui5.Title>
      </ui5.Page>
      <ui5.Bar
        design="Header"
        endContent={<span>SAP BTP, Kyma runtime</span>}
        startContent={<span>Select your credentials:</span>}
      >
        <Secrets handler={(e: any) => handler(e)} style={{ width: "100vw" }} />
      </ui5.Bar>
      <>
          <div>
              <ui5.FlexBox
                  style={{
                      height: "90vh",
                      width: "100%",
                  }}
              >
                  <ui5.SideNavigation
                      style={{
                          width: "30%",
                          height: "90vh",
                      }}
                  >
                      <ui5.SideNavigationItem
                          text="Marketplace"
                          icon="home"
                          onClick={() => {
                              setPageContent(<ServiceOfferings secret={secret}/>);
                          }}
                      />
                      <ui5.SideNavigationItem
                          text="Service Instances"
                          icon="home"
                          onClick={() => {
                              setPageContent(<ServiceInstancesView/>);
                          }}
                      />
                  </ui5.SideNavigation>
                  <ui5.Page
                      backgroundDesign="Solid"
                      style={{
                          height: "90vh",
                          width: "70%",
                      }}
                  >
                      {pageContent}
                  </ui5.Page>
              </ui5.FlexBox>
          </div>

      </>
    </>
  );
}

export default Overview;

import ServiceOfferings from "./ServiceOfferings";
import ServiceInstances from "./ServiceInstances";
import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./Secrets";
import React from "react";

function Overview(props: any) {
  const [secret, setSecret] = React.useState(null);
  function handler(e: any) {
    setSecret(e)
  }
  
  return (
      <>
        <div>
          <ui5.Page
              backgroundDesign="Solid"
              header={<ui5.Bar design="Header">
                <ui5.Label>Service Management</ui5.Label>
              </ui5.Bar>}
              footer={<ui5.Bar design="Footer">
                <ui5.Label>Footer</ui5.Label>
              </ui5.Bar>}
              style={{
                height: '100vh'
              }}
          >
            <ui5.Grid>
              <div data-layout-indent="XL12" data-layout-span="XL12">
                <Secrets handler={(e :any) => handler(e)}/>
              </div>
              <div data-layout-indent="XL12" data-layout-span="XL12">
              <ServiceOfferings secret={secret}/>
              </div>
              <div data-layout-indent="XL12" data-layout-span="XL12">
                <ServiceInstances/>
              </div>
            </ui5.Grid>
          </ui5.Page>
        </div>
      </>
  )
}

export default Overview;

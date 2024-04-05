import ServiceOfferings from "./ServiceOfferings";
import ServiceInstances from "./ServiceInstances";
import * as ui5 from "@ui5/webcomponents-react";
import Secrets from "./Secrets";
import React from "react";

function Overview(props) {
  const [secret, setSecret] = React.useState(null);

  function get() {
    let b
    if (secret !== "") {
      console.log(`sending secret to ServiceOfferings: ${secret}`)
      b = <ServiceOfferings secret={secret}/>
      console.log("secret found")
    }

    return b;
  }

  function handler(e) {
    console.log("handler")
    console.log(e)
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
                <Secrets handler={(e) => handler(e)}/>
              </div>
              {get()}
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

import * as ui5 from "@ui5/webcomponents-react";

import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';
import { ApiError } from "../shared/models";
import { useEffect, useState } from "react";

interface StatusMessageProps {
    error: ApiError | undefined;
    success: string | undefined;
}

function StatusMessage(props: StatusMessageProps) {

    const [message, setMessage] = useState("");


    useEffect(() => {

        if (props.error) {
            var message = props.error!!.name + " - " + props.error!!.code + " - " + props.error!!.message;

            if (props.error!!.response) {
                message += " - " + props.error!!.response.data;
                setMessage(message);
            }

        }
    }, [props.error, props.success]);


    const renderData = () => {

        if (props.error) {
            return (
                <ui5.FormItem>
                    <ui5.MessageStrip
                        design="Negative"
                        onClose={function _a() { }}>
                        {message}
                    </ui5.MessageStrip>
                </ui5.FormItem>
            );
        } else if (props.success) {
            return (
                <ui5.FormItem>
                    <ui5.MessageStrip
                        design="Information"
                        onClose={function _a() { }}>
                        {props.success}
                    </ui5.MessageStrip>
                </ui5.FormItem>
            );
        }
    };

    return <>{renderData()}</>;
}

export default StatusMessage;
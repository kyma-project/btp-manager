function Ok(value: any)  {
    if (value == null) {
        return false
    }

    if (value === undefined) {
        return false
    }

    if (!value) {
        return false
    }

    if ( typeof value == 'string' && value === "") {
        return false
    }

    if (Array.isArray(value)) {
        if (value.length === 0) {
            return false
        }
    }
    return true
}

export default Ok;
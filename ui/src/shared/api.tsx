function api(url :string) {
    return `http://localhost:3001/backend/api/v1/namespaces/kyma-system/services/btp-manager-metrics-service:8080/proxy/api/${url}`
}

export default api;
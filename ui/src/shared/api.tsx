function api(url :string) {
    const port = process.env.BTP_MANAGER_API_PORT
    return `http://localhost:${port}/api/${url}`
}

export default api ;
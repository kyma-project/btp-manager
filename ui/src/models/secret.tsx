export default interface Secrets {
    items: Secret[];
}

interface Secret {
    name: string;
    namespace: string;
}
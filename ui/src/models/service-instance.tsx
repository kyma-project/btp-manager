export interface ServiceInstances {
    items: ServiceInstance[]
}

interface ServiceInstance
{
    id: string;
    name: string;
    context: string;
    namespace: string;
    service_bindings: ServiceInstaceBindings[];
}

interface ServiceInstaceBindings 
{
    id: string;
    name: string;
    namespace: string;
}
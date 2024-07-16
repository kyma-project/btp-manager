package responses

import (
	"testing"

	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"github.com/stretchr/testify/assert"
)


func TestConverters(t *testing.T) {

    t.Run("should set correct len of items", func(t *testing.T) {
        // given
        servicesInstances := &types.ServiceInstances{
            Items: []types.ServiceInstance{
                {
                    Common: types.Common{
                        ID:          "1",
                        Name:        "service-1",
                        Description: "",
                    },
                },
                {
                    Common: types.Common{
                        ID:          "2",
                        Name:        "service-2",
                        Description: "",
                    },
                },
            },
        } 
        // when 
        vmServiceInstances := ToServiceInstancesVM(servicesInstances)

        // then
        assert.Equal(t, 2, vmServiceInstances.NumItems)
    })
}
package registry

import (
	"testing"

	"github.com/tj/assert"
)

func TestGetPartition(t *testing.T) {
	a := assert.New(t)

	cases := []Metadata{
		{
			"zone": "zone01",
		},
		{
			"zone": "RZ01",
		},
		{
			"zone": "test_01",
		},
		{
			"zone": "foo01",
		},
		{
			"zone": "foo02test01",
		},
	}

	cases1 := []Metadata{
		{
			"zone": "foo02test",
		},
		{},
		{
			"zone": "test_rt",
		},
	}

	cases00 := []Metadata{
		{
			"zone": "RZ00",
		},
		{
			"zone": "RZ01",
		},
		{
			"zone": "RZ02",
		},
		{
			"zone": "RZ03",
		},
		{
			"zone": "RZ04",
		},
		{
			"zone": "RZ05",
		},
		{
			"zone": "RZ06",
		},
		{
			"zone": "RZ07",
		},
		{
			"zone": "RZ08",
		},
		{
			"zone": "RZ09",
		},
		{
			"zone": "RZ10",
		},
		{
			"zone": "RZ11",
		},
	}

	cases01 := []Metadata{
		{
			"create_time":      "1640834439570",
			"zone":             "RZ09",
			"gRPC_port":        "9090",
			"warmup":           "30000",
			"last_modify_time": "1640834439570",
		},
	}

	for _, md := range cases01 {
		//t.Log(md.GetPartition())
		a.Equal(md.GetPartition(), 9)
	}

	for _, md := range cases {
		//t.Log(md.GetPartition())
		a.Equal(md.GetPartition(), 1)
	}

	for i, md := range cases00 {
		//t.Log(md.GetPartition())
		a.Equal(md.GetPartition(), i)
	}

	for _, md := range cases1 {
		//t.Log(md.GetPartition())
		a.Equal(md.GetPartition(), -1)
	}
}

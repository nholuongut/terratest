package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoute53Record(t *testing.T) {
	t.Parallel()
	region := GetRandomStableRegion(t, nil, nil)
	c, err := NewRoute53ClientE(t, region)
	require.NoError(t, err)

	domain := fmt.Sprintf("terratest%dexample.com", time.Now().UnixNano())
	hostedZone, err := c.CreateHostedZone(&route53.CreateHostedZoneInput{
		Name:            aws.String(domain),
		CallerReference: aws.String(fmt.Sprint(time.Now().UnixNano())),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := c.DeleteHostedZone(&route53.DeleteHostedZoneInput{
			Id: hostedZone.HostedZone.Id,
		})
		require.NoError(t, err)
	})

	recordName := fmt.Sprintf("record.%s", domain)
	resourceRecordSet := &route53.ResourceRecordSet{
		Name: &recordName,
		Type: aws.String("A"),
		TTL:  aws.Int64(60),
		ResourceRecords: []*route53.ResourceRecord{
			{
				Value: aws.String("127.0.0.1"),
			},
		},
	}
	_, err = c.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZone.HostedZone.Id,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action:            proto.String("CREATE"),
					ResourceRecordSet: resourceRecordSet,
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := c.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: hostedZone.HostedZone.Id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action:            proto.String("DELETE"),
						ResourceRecordSet: resourceRecordSet,
					},
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("ExistingRecord", func(t *testing.T) {
		route53Record := GetRoute53Record(t, *hostedZone.HostedZone.Id, recordName, *resourceRecordSet.Type, region)
		require.NotNil(t, route53Record)
		assert.Equal(t, recordName+".", *route53Record.Name)
		assert.Equal(t, *resourceRecordSet.Type, *route53Record.Type)
		assert.Equal(t, "127.0.0.1", *route53Record.ResourceRecords[0].Value)
	})

	t.Run("NotExistRecord", func(t *testing.T) {
		route53Record, err := GetRoute53RecordE(t, *hostedZone.HostedZone.Id, "ne"+recordName, "A", region)
		assert.Error(t, err)
		assert.Nil(t, route53Record)
	})

}

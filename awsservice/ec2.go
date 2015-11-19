package awsservice

import (
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// AWS SDK pointer-itis
var True = true
var False = false

type InstancesDefinition struct {
	AMI           string
	Subnet        string
	SecurityGroup string
	Keypair       string
	Type          string
	UserData      []byte
	Count         int
	RootSizeGB    int // Optional (default: 20)
}

type InstanceInfo struct {
	AMI            string
	Keypair        string
	Type           string
	ID             string
	PrivateIP      string
	PublicIP       string
	Subnet         string
	SecurityGroups []string
	Tags           map[string]string
}

type SubnetInfo struct {
	AvailabilityZone     string
	AvailableIPAddresses int64
	CIDR                 string
	State                string
	ID                   string
	Tags                 map[string]string
	VPC                  string
}

func (aws *RealAWSService) RunInstances(idef *InstancesDefinition) ([]string, error) {
	count := int64(idef.Count)
	rs := int64(20)
	vt := "gp2"
	rdn := "/dev/xvda"
	ud := base64.StdEncoding.EncodeToString(idef.UserData)
	if idef.RootSizeGB != 0 {
		rs = int64(idef.RootSizeGB)
	}
	bdm := ec2.BlockDeviceMapping{
		DeviceName: &rdn,
		Ebs: &ec2.EbsBlockDevice{
			DeleteOnTermination: &True,
			VolumeSize:          &rs,
			VolumeType:          &vt,
		},
	}
	ri := ec2.RunInstancesInput{
		ImageId:             &idef.AMI,
		MinCount:            &count,
		MaxCount:            &count,
		KeyName:             &idef.Keypair,
		InstanceType:        &idef.Type,
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{&bdm},
		SecurityGroupIds:    []*string{&idef.SecurityGroup},
		SubnetId:            &idef.Subnet,
		UserData:            &ud,
	}
	r, err := aws.ec2.RunInstances(&ri)
	if err != nil {
		return []string{}, err
	}
	instances := []string{}
	for _, inst := range r.Instances {
		instances = append(instances, *(inst.InstanceId))
	}
	return instances, nil
}

func (aws *RealAWSService) StartInstances(ids []string) error {
	si := ec2.StartInstancesInput{
		InstanceIds: stringSlicetoStringPointerSlice(ids),
	}
	_, err := aws.ec2.StartInstances(&si)
	return err
}

func (aws *RealAWSService) StopInstances(ids []string) error {
	si := ec2.StopInstancesInput{
		InstanceIds: stringSlicetoStringPointerSlice(ids),
	}
	_, err := aws.ec2.StopInstances(&si)
	return err
}

func (aws *RealAWSService) FindInstancesByTag(n string, v string) ([]string, error) {
	fn := fmt.Sprintf("tag:%v", n)
	f := ec2.Filter{
		Name:   &fn,
		Values: []*string{&v},
	}
	dii := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{&f},
	}
	instances := []string{}
	r, err := aws.ec2.DescribeInstances(&dii)
	if err != nil {
		return instances, err
	}
	for _, rev := range r.Reservations {
		for _, inst := range rev.Instances {
			instances = append(instances, *(inst.InstanceId))
		}
	}
	return instances, nil
}

func (aws *RealAWSService) TagInstances(ids []string, n string, v string) error {
	tag := ec2.Tag{
		Key:   &n,
		Value: &v,
	}
	cti := ec2.CreateTagsInput{
		Tags:      []*ec2.Tag{&tag},
		Resources: stringSlicetoStringPointerSlice(ids),
	}
	_, err := aws.ec2.CreateTags(&cti)
	return err
}

func (aws *RealAWSService) DeleteTag(ids []string, n string) error {
	tag := ec2.Tag{
		Key: &n,
	}
	dti := ec2.DeleteTagsInput{
		Tags:      []*ec2.Tag{&tag},
		Resources: stringSlicetoStringPointerSlice(ids),
	}
	_, err := aws.ec2.DeleteTags(&dti)
	return err
}

func (aws *RealAWSService) GetSubnetInfo(id string) (*SubnetInfo, error) {
	result := &SubnetInfo{}
	dsi := ec2.DescribeSubnetsInput{
		SubnetIds: stringSlicetoStringPointerSlice([]string{id}),
	}
	res, err := aws.ec2.DescribeSubnets(&dsi)
	if err != nil {
		return result, err
	}
	result.AvailabilityZone = *res.Subnets[0].AvailabilityZone
	result.AvailableIPAddresses = *res.Subnets[0].AvailableIpAddressCount
	result.CIDR = *res.Subnets[0].CidrBlock
	result.State = *res.Subnets[0].State
	result.ID = *res.Subnets[0].SubnetId
	result.VPC = *res.Subnets[0].VpcId
	tags := map[string]string{}
	for _, t := range res.Subnets[0].Tags {
		tags[*t.Key] = *t.Value
	}
	result.Tags = tags
	return result, nil
}

func (aws *RealAWSService) GetInstancesInfo(ids []string) ([]InstanceInfo, error) {
	result := []InstanceInfo{}
	dii := ec2.DescribeInstancesInput{
		InstanceIds: stringSlicetoStringPointerSlice(ids),
	}
	res, err := aws.ec2.DescribeInstances(&dii)
	if err != nil {
		return result, err
	}
	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			ii := InstanceInfo{
				AMI:       *i.ImageId,
				Keypair:   *i.KeyName,
				Type:      *i.InstanceType,
				ID:        *i.InstanceId,
				PublicIP:  *i.PublicIpAddress,
				PrivateIP: *i.PrivateIpAddress,
				Subnet:    *i.SubnetId,
			}
			sgl := []string{}
			for _, sg := range i.SecurityGroups {
				sgl = append(sgl, *sg.GroupId)
			}
			tags := map[string]string{}
			for _, t := range i.Tags {
				tags[*t.Key] = *t.Value
			}
			ii.SecurityGroups = sgl
			ii.Tags = tags
			result = append(result, ii)
		}
	}
	return result, nil
}

func (aws *RealAWSService) TerminateInstances(ids []string) error {
	tii := ec2.TerminateInstancesInput{
		InstanceIds: stringSlicetoStringPointerSlice(ids),
	}
	_, err := aws.ec2.TerminateInstances(&tii)
	return err
}

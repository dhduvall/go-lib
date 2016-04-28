package awsservice

import (
	"bytes"
	"compress/gzip"
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
	GetPublicIP   bool
	UserData      []byte
	Count         int
	RootSizeGB    int // Optional (default: 20)
}

type InstanceInfo struct {
	AMI                string
	Keypair            string
	Type               string
	ID                 string
	PrivateIP          string
	PublicIP           string
	Subnet             string
	SecurityGroups     []string
	State              string
	StateReasonCode    string
	StateReasonMessage string
	Tags               map[string]string
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

func encodeUserData(ud []byte) (string, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(ud); err != nil {
		return "", err
	}
	if err := w.Flush(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (aws *RealAWSService) RunInstances(idef *InstancesDefinition) ([]string, error) {
	count := int64(idef.Count)
	rs := int64(20)
	vt := "gp2"
	rdn := "/dev/xvda"
	ud, err := encodeUserData(idef.UserData)
	if err != nil {
		return []string{}, err
	}
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
		UserData:            &ud,
	}
	if idef.GetPublicIP {
		devindx := int64(0)
		ri.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{&ec2.InstanceNetworkInterfaceSpecification{
			AssociatePublicIpAddress: &True,
			Groups:      []*string{&idef.SecurityGroup},
			DeviceIndex: &devindx,
			SubnetId:    &idef.Subnet,
		}}
	} else {
		ri.SubnetId = &idef.Subnet
		ri.SecurityGroupIds = []*string{&idef.SecurityGroup}
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
			instances = append(instances, drefStringPtr(inst.InstanceId))
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
	result.AvailabilityZone = drefStringPtr(res.Subnets[0].AvailabilityZone)
	result.AvailableIPAddresses = drefInt64Ptr(res.Subnets[0].AvailableIpAddressCount)
	result.CIDR = drefStringPtr(res.Subnets[0].CidrBlock)
	result.State = drefStringPtr(res.Subnets[0].State)
	result.ID = drefStringPtr(res.Subnets[0].SubnetId)
	result.VPC = drefStringPtr(res.Subnets[0].VpcId)
	tags := map[string]string{}
	for _, t := range res.Subnets[0].Tags {
		tags[drefStringPtr(t.Key)] = drefStringPtr(t.Value)
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
				AMI:       drefStringPtr(i.ImageId),
				Keypair:   drefStringPtr(i.KeyName),
				Type:      drefStringPtr(i.InstanceType),
				ID:        drefStringPtr(i.InstanceId),
				PrivateIP: drefStringPtr(i.PrivateIpAddress),
				Subnet:    drefStringPtr(i.SubnetId),
				PublicIP:  drefStringPtr(i.PublicIpAddress),
				State:     drefStringPtr(i.State.Name),
			}
			if i.StateReason != nil {
				ii.StateReasonCode = drefStringPtr(i.StateReason.Code)
				ii.StateReasonMessage = drefStringPtr(i.StateReason.Message)
			}
			sgl := []string{}
			for _, sg := range i.SecurityGroups {
				sgl = append(sgl, drefStringPtr(sg.GroupId))
			}
			tags := map[string]string{}
			for _, t := range i.Tags {
				tags[drefStringPtr(t.Key)] = drefStringPtr(t.Value)
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

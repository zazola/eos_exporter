package eosclient

// Used to run eos commands.
// This code can be vastly improved.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"strings"
	"syscall"
	"unicode"

	"github.com/cernbox/reva/api"
	"go.uber.org/zap"
)


type Options struct {
	// Location of the eos binary. Default is /usr/bin/eos.
	EosBinary string

	// Location of the xrdcopy binary. Default is /usr/bin/xrdcopy.
	XrdcopyBinary string

	// URL of the EOS MGM. Default is root://eos-test.org
	URL string

	// Location on the local fs where to store reads. Defaults to os.TempDir()
	CacheDirectory string

	// Enables logging of the commands executed. Defaults to false
	EnableLogging bool

	// Logger to use
	Logger *zap.Logger
}

func (opt *Options) init() {
	if opt.EosBinary == "" {
		opt.EosBinary = "/usr/bin/eos"
	}

	if opt.XrdcopyBinary == "" {
		opt.XrdcopyBinary = "/usr/bin/xrdcopy"
	}

	if opt.URL == "" {
		opt.URL = "root://eos-example.org"
	}

	if opt.CacheDirectory == "" {
		opt.CacheDirectory = os.TempDir()
	}

	if opt.Logger == nil {
		l, _ := zap.NewProduction()
		opt.Logger = l
	}
}

// Client performs actions against a EOS management node (MGM).
// It requires the eos-client and xrootd-client packages installed to work.
type Client struct {
	opt *Options
}

type NodeInfo struct {
	Hostport              string
	Status                string
	Nofs                  string
	SumStatStatfsFree	  string
	SumStatStatfsUsed	  string
	SumStatStatfsTotal    string
	SumStatStatFilesFree  string
	SumStatStatFilesUsed  string
	SumStatStatFilesTotal string
	SumStatRopen		  string
	SumStatWopen		  string
	CfgStatSysThreads	  string
	SumStatNetInratemib	  string
	SumStatNetOutratemib  string
}

type SpaceInfo struct {
	Type							    string
	Name								string
	CfgGroupSize 						string
	CfgGroupMod			  				string
	Nofs	  							string
	AvgStatDiskLoad 	    			string
	SigStatDiskLoad  					string
	SumStatDiskReadratemb  				string
	SumStatDiskWriteratemb 				string
	SumStatNetEthratemib		  		string
	SumStatNetInratemib		  			string
	SumStatNetOutratemib	  			string
	SumStatRopen	  					string
	SumStatWopen  						string
	SumStatStatfsUsedbytes 				string
	SumStatStatfsFreebytes 				string
	SumStatStatfsCapacity 				string
	SumStatUsedfiles 					string
	SumStatStatfsFfiles 				string
	SumStatStatfsFiles 					string
	SumStatStatfsCapacityConfigstatusRw string
	SumNofsConfigstatusRw 				string
	CfgQuota 							string
	CfgNominalsize 						string
	CfgBalancer 						string
	CfgBalancerThreshold 				string
	SumStatBalancerRunning 				string
	SumStatDrainerRunning 				string
	SumStatDiskIopsConfigstatusRw 		string
	SumStatDiskBwConfigstatusRw 		string
}

type GroupInfo struct {
	Name 				   string
	CfgStatus 			   string
	Nofs 				   string
	AvgStatDiskLoad 	   string
	SigStatDiskLoad        string
	SumStatDiskReadratemb  string
	SumStatDiskWriteratemb string
	SumStatNetEthratemib   string
	SumStatNetInratemib    string
	SumStatNetOutratemib   string
	SumStatRopen      	   string
	SumStatWopen 		   string
	SumStatStatfsUsedbytes string
	SumStatStatfsFreebytes string
	SumStatStatfsCapacity  string
	SumStatUsedfiles       string
	SumStatStatfsFfree     string
	SumStatStatfsFiles     string
	DevStatStatfsFilled    string
	AvgStatStatfsFilled    string
	SigStatStatfsFilled    string
	CfgStatBalancing       string
	SumStatBalancerRunning string
	SumStatDrainerRunning  string
}

type FSInfo struct {
	Host 					   string
	Port 					   string
	Id 						   string
	Uuid 					   string
	Path 					   string
	Schedgroup 				   string
	StatBoot 				   string
	Configstatus 			   string
	Headroom 				   string
	StatErrc 				   string
	StatErrmsg 				   string
	StatDiskLoad               string
	StatDiskReadratemb         string
	StatDiskWriteratemb        string
	StatNetEthratemib          string
	StatNetInratemib           string
	StatNetOutratemib          string
	StatRopen                  string
	StatWopen                  string
	StatStatfsFreebytes 	   string
	StatStatfsUsedbytes 	   string
	StatStatfsCapacity 		   string
	StatUsedfiles              string
	StatStatfsFfree 		   string
	StatStatfsFused            string
	StatStatfsFiles            string
	Drainstatus                string
	StatDrainprogress          string
	StatDrainfiles             string
	StatDrainbytesleft         string
	StatDrainretry             string
	StatDrainFailed            string
	Graceperiod                string
	StatTimeleft               string
	StatActive                 string
	StatBalancerRunning        string
	StatDrainerRunning         string
	StatDiskIops               string
	StatDiskBw                 string
	StatGeotag                 string
	StatHealth                 string
	StatHealthRedundancyFactor string
	StatHealthDrivesFailed     string
	StatHealthDrivesTotal      string
	StatHealthIndicator        string
}

func New(opt *Options) (*Client, error) {
	opt.init()
	c := new(Client)
	c.opt = opt
	return c, nil
}

func getUnixUser(username string) (*osuser.User, error) {
	return osuser.Lookup(username)
}

// exec executes the command and returns the stdout, stderr and return code
func (c *Client) execute(cmd *exec.Cmd) (string, string, error) {
	cmd.Env = []string{
		"EOS_MGM_URL=" + c.opt.URL,
	}

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	if c.opt.EnableLogging {
		c.opt.Logger.Info("eosclient", zap.String("cmd", fmt.Sprintf("%+v", cmd)))
	}

	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			switch status.ExitStatus() {
			case 2:
				err = api.NewError(api.StorageNotFoundErrorCode)
			}
		}
	}
	return outBuf.String(), errBuf.String(), err
}

// List the nodes on the cluster
func (c *Client) ListNode(ctx context.Context, username string) ([]*NodeInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "node", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseNodesInfo(stdout)
}

// List the spaces on the cluster
func (c *Client) ListSpace(ctx context.Context, username string) ([]*SpaceInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "space", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseSpacesInfo(stdout)
}

// List the scheduling groups on the cluster
func (c *Client) ListGroup(ctx context.Context, username string) ([]*GroupInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "group", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseGroupsInfo(stdout)
}

// List the filesystems on the cluster
func (c *Client) ListFS(ctx context.Context, username string) ([]*FSInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "fs", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseFSsInfo(stdout)
}

// Convert a monitoring format line into a map
func getMap(line string) map[string]string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)

		}
	}

	// splitting string by space but considering quoted section
	items := strings.FieldsFunc(line, f)

	// create and fill the map
	m := make(map[string]string)
	for _, item := range items {
		x := strings.Split(item, "=")
		m[x[0]] = x[1]
	}
	return m

}

// Gathers information of all nodes
func (c *Client) parseNodesInfo(raw string) ([]*NodeInfo, error) {
	fstinfos := []*NodeInfo{}
	rawLines := strings.Split(raw, "\n")
	for _, rl := range rawLines {
		if rl == "" {
			continue
		}
		node, err := c.parseNodeInfo(rl)

		if err != nil {
			return nil, err
		}
		fstinfos = append(fstinfos, node)
	}
	return fstinfos, nil
}

// Gathers information of one single node
func (c *Client) parseNodeInfo(line string) (*NodeInfo, error) {
	//kv := make(map[string]string)
	kv := getMap(line)
	fst := &NodeInfo{
		Hostport: 				kv["hostport"],
		Status:   				kv["status"],
		Nofs:     				kv["nofs"],
		SumStatStatfsFree:		kv["sum.stat.statfs.freebytes"],
		SumStatStatfsUsed:	    kv["sum.stat.statfs.usedbytes"],
		SumStatStatfsTotal: 	kv["sum.stat.statfs.capacity"],
		SumStatStatFilesFree:   kv["sum.stat.statfs.ffree"],
		SumStatStatFilesUsed:   kv["sum.stat.usedfiles"],
		SumStatStatFilesTotal:  kv["sum.stat.statfs.files"],
		SumStatRopen:		    kv["sum.stat.ropen"],
		SumStatWopen:		    kv["sum.stat.wopen"],
		CfgStatSysThreads:	    kv["cfg.stat.sys.threads"],
		SumStatNetInratemib:	kv["sum.stat.net.inratemib"],
		SumStatNetOutratemib:   kv["sum.stat.net.outratemib"],
	}
	return fst, nil
}


// Gathers the information of all spaces.
func (c *Client) parseSpacesInfo(raw string) ([]*SpaceInfo, error) {
	spaceinfos := []*SpaceInfo{}
	rawLines := strings.Split(raw, "\n")
	for _, rl := range rawLines {
		if rl == "" {
			continue
		}
		space, err := c.parseSpaceInfo(rl)

		if err != nil {
			return nil, err
		}
		spaceinfos = append(spaceinfos, space)
	}
	return spaceinfos, nil
}

// Gathers information of one single space
func (c *Client) parseSpaceInfo(line string) (*SpaceInfo, error) {
	//kv := make(map[string]string)
	kv := getMap(line)
	space := &SpaceInfo{
		kv["type"],
		kv["name"],
		kv["cfg.groupsize"],
		kv["cfg.groupmod"],
		kv["nofs"],
		kv["avg.stat.disk.load"],
		kv["sig.stat.disk.load"],
		kv["sum.stat.disk.readratemb"],
		kv["sum.stat.disk.writeratemb"],
		kv["sum.stat.net.ethratemib"],
		kv["sum.stat.net.inratemib"],
		kv["sum.stat.net.outratemib"],
		kv["sum.stat.ropen"],
		kv["sum.stat.wopen"],
		kv["sum.stat.statfs.usedbytes"],
		kv["sum.stat.statfs.freebytes"],
		kv["sum.stat.statfs.capacity"],
		kv["sum.stat.usedfiles"],
		kv["sum.stat.statfs.ffiles"],
		kv["sum.stat.statfs.files"],
		kv["sum.stat.statfs.capacity?configstatus@rw"],
		kv["sum.<n>?configstatus@rw"],
		kv["cfg.quota"],
		kv["cfg.nominalsize"],
		kv["cfg.balancer"],
		kv["cfg.balancer.threshold"],
		kv["sum.stat.balancer.running"],
		kv["sum.stat.drainer.running"],
		kv["sum.stat.disk.iops?configstatus@rw"],
		kv["sum.stat.disk.bw?configstatus@rw"],
	}
	return space, nil
}

// Gathers information of all groups
func (c *Client) parseGroupsInfo(raw string) ([]*GroupInfo, error) {
	groupinfos := []*GroupInfo{}
	rawLines := strings.Split(raw, "\n")
	for _, rl := range rawLines {
		if rl == "" {
			continue
		}
		group, err := c.parseGroupInfo(rl)

		if err != nil {
			return nil, err
		}
		groupinfos = append(groupinfos, group)
	}
	return groupinfos, nil
}

// Gathers information of one single group
func (c *Client) parseGroupInfo(line string) (*GroupInfo, error) {
	//kv := make(map[string]string)
	kv := getMap(line)
	group := &GroupInfo{
		kv["name"],
		kv["cfg.status"],
		kv["nofs"],
		kv["avg.stat.disk.load"],
		kv["sig.stat.disk.load"],
		kv["sum.stat.disk.readratemb"],
		kv["sum.stat.disk.writeratemb"],
		kv["sum.stat.net.ethratemib"],
		kv["sum.stat.net.inratemib"],
		kv["sum.stat.net.outratemib"],
		kv["sum.stat.ropen"],
		kv["sum.stat.wopen"],
		kv["sum.stat.statfs.usedbytes"],
		kv["sum.stat.statfs.freebytes"],
		kv["sum.stat.statfs.capacity"],
		kv["sum.stat.usedfiles"],
		kv["sum.stat.statfs.ffree"],
		kv["sum.stat.statfs.files"],
		kv["dev.stat.statfs.filled"],
		kv["avg.stat.statfs.filled"],
		kv["sig.stat.statfs.filled"],
		kv["cfg.stat.balancing"],
		kv["sum.stat.balancer.running"],
		kv["sum.stat.drainer.running"],
	}
	return group, nil
}

// Gathers information of all filesystems
func (c *Client) parseFSsInfo(raw string) ([]*FSInfo, error) {
	fsinfos := []*FSInfo{}
	rawLines := strings.Split(raw, "\n")
	for _, rl := range rawLines {
		if rl == "" {
			continue
		}
		fs, err := c.parseFSInfo(rl)

		if err != nil {
			return nil, err
		}
		fsinfos = append(fsinfos, fs)
	}
	return fsinfos, nil
}

// Gathers information of one single filesystem
func (c *Client) parseFSInfo(line string) (*FSInfo, error) {
	//kv := make(map[string]string)
	kv := getMap(line)
	fs := &FSInfo{
		kv["host"],
		kv["port"],
		kv["id"],
		kv["uuid"],
		kv["path"],
		kv["schedgroup"],
		kv["stat.boot"],
		kv["configstatus"],
		kv["headroom"],
		kv["stat.errc"],
		kv["stat.errmsg"],
		kv["stat.disk.load"],
		kv["stat.disk.readratemb"],
		kv["stat.disk.writeratemb"],
		kv["stat.net.ethratemib"],
		kv["stat.net.inratemib"],
		kv["stat.net.outratemib"],
		kv["stat.ropen"],
		kv["stat.wopen"],
		kv["stat.statfs.freebytes"],
		kv["stat.statfs.usedbytes"],
		kv["stat.statfs.capacity"],
		kv["stat.usedfiles"],
		kv["stat.statfs.ffree"],
		kv["stat.statfs.fused"],
		kv["stat.statfs.files"],
		kv["drainstatus"],
		kv["stat.drainprogress"],
		kv["stat.drainfiles"],
		kv["stat.drainbytesleft"],
		kv["stat.drainretry"],
		kv["stat.drain.failed"],
		kv["graceperiod"],
		kv["stat.timeleft"],
		kv["stat.active"],
		kv["stat.balancer.running"],
		kv["stat.drainer.running"],
		kv["stat.disk.iops"],
		kv["stat.disk.bw"],
		kv["stat.geotag"],
		kv["stat.health"],
		kv["stat.health.redundancy_factor"],
		kv["stat.health.drives_failed"],
		kv["stat.health.drives_total"],
		kv["stat.health.indicator"],
	}
	return fs, nil
}
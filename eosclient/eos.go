package eosclient

// Used to run eos commands.
// This code can be vastly improved.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	// "github.com/cernbox/reva/api"
	"time"

	"go.uber.org/zap"
)

var cmdTimeout = 10 * time.Second // Time-out for the EOS commands

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
	SumStatStatfsFree     string
	SumStatStatfsUsed     string
	SumStatStatfsTotal    string
	SumStatStatFilesFree  string
	SumStatStatFilesUsed  string
	SumStatStatFilesTotal string
	SumStatRopen          string
	SumStatWopen          string
	CfgStatSysThreads     string
	SumStatNetInratemib   string
	SumStatNetOutratemib  string
}

type SpaceInfo struct {
	Type                                string
	Name                                string
	CfgGroupSize                        string
	CfgGroupMod                         string
	Nofs                                string
	AvgStatDiskLoad                     string
	SigStatDiskLoad                     string
	SumStatDiskReadratemb               string
	SumStatDiskWriteratemb              string
	SumStatNetEthratemib                string
	SumStatNetInratemib                 string
	SumStatNetOutratemib                string
	SumStatRopen                        string
	SumStatWopen                        string
	SumStatStatfsUsedbytes              string
	SumStatStatfsFreebytes              string
	SumStatStatfsCapacity               string
	SumStatUsedfiles                    string
	SumStatStatfsFfiles                 string
	SumStatStatfsFiles                  string
	SumStatStatfsCapacityConfigstatusRw string
	SumNofsConfigstatusRw               string
	CfgQuota                            string
	CfgNominalsize                      string
	CfgBalancer                         string
	CfgBalancerThreshold                string
	SumStatBalancerRunning              string
	SumStatDrainerRunning               string
	SumStatDiskIopsConfigstatusRw       string
	SumStatDiskBwConfigstatusRw         string
}

type GroupInfo struct {
	Name                   string
	CfgStatus              string
	Nofs                   string
	AvgStatDiskLoad        string
	SigStatDiskLoad        string
	SumStatDiskReadratemb  string
	SumStatDiskWriteratemb string
	SumStatNetEthratemib   string
	SumStatNetInratemib    string
	SumStatNetOutratemib   string
	SumStatRopen           string
	SumStatWopen           string
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
	Host                       string
	Port                       string
	Id                         string
	Uuid                       string
	Path                       string
	Schedgroup                 string
	StatBoot                   string
	Configstatus               string
	Headroom                   string
	StatErrc                   string
	StatErrmsg                 string
	StatDiskLoad               string
	StatDiskReadratemb         string
	StatDiskWriteratemb        string
	StatNetEthratemib          string
	StatNetInratemib           string
	StatNetOutratemib          string
	StatRopen                  string
	StatWopen                  string
	StatStatfsFreebytes        string
	StatStatfsUsedbytes        string
	StatStatfsCapacity         string
	StatUsedfiles              string
	StatStatfsFfree            string
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

type VSInfo struct {
	EOSmgm    string
	Hostname  string
	Port      string
	Geotag    string
	Vsize     string
	Rss       string
	Threads   string
	Sockets   string
	EOSfst    string
	Xrootdfst string
	KernelV   string
	Start     string
	Uptime    string
}

type NSInfo struct {
	Boot_file_time                             string
	Boot_status                                string
	Boot_time                                  string
	Cache_container_maxsize                    string
	Cache_container_occupancy                  string
	Cache_files_maxsize                        string
	Cache_files_occupancy                      string
	Fds_all                                    string
	Fusex_activeclients                        string
	Fusex_caps                                 string
	Fusex_clients                              string
	Fusex_lockedclients                        string
	Latency_dirs                               string
	Latency_files                              string
	Latency_pending_updates                    string
	Latencypeak_eosviewmutex_1min              string
	Latencypeak_eosviewmutex_2min              string
	Latencypeak_eosviewmutex_5min              string
	Latencypeak_eosviewmutex_last              string
	Memory_growth                              string
	Memory_resident                            string
	Memory_share                               string
	Memory_virtual                             string
	Stat_threads                               string
	Total_directories                          string
	Total_directories_changelog_avg_entry_size string
	Total_directories_changelog_size           string
	Total_files                                string
	Total_files_changelog_avg_entry_size       string
	Total_files_changelog_size                 string
	Uptime                                     string
}

type NSActivityInfo struct {
	User       string
	Gid        string
	Operation  string
	Sum        string
	Last_5s    string
	Last_60s   string
	Last_300s  string
	Last_3600s string
	Exec       string
	Sigma      string
	Exec99     string
	Max        string
}

type Sys struct {
	Eos struct {
		Start   string `json:"start"`
		Version string `json:"version"`
	} `json:"eos"`
	Kernel  string `json:"kernel"`
	Rss     int    `json:"rss"`
	Sockets int    `json:"sockets"`
	Threads int    `json:"threads"`
	Uptime  string `json:"uptime"`
	Vsize   int    `json:"vsize"`
	Xrootd  struct {
		Version string `json:"version"`
	} `json:"xrootd"`
}

type Stat struct {
	Geotag string `json:"geotag"`
	Sys    Sys    `json:"sys"`
}

type NodeLSCfg struct {
	Stat Stat `json:"stat"`
}

type NodeLS struct {
	HostPort string     `json:"hostport"` // "hostname:port"
	Cfg      *NodeLSCfg `json:"cfg"`
}

type NodeLSResponse struct {
	ErrorMsg string    `json:"errormsg"`
	Result   []*NodeLS `json:"result"`
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
				err = fmt.Errorf("error: FIXME") // api.NewError(api.StorageNotFoundErrorCode)
			}
		}
	}
	return outBuf.String(), errBuf.String(), err
}

// List the nodes on the instance
func (c *Client) ListNode(ctx context.Context, username string) ([]*NodeInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}
	var (
		ctxWt  context.Context
		cancel context.CancelFunc
	)

	ctxWt, cancel = context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWt, "/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "node", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseNodesInfo(stdout)
}

// List the spaces on the instance
func (c *Client) ListSpace(ctx context.Context, username string) ([]*SpaceInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}

	var (
		ctxWt  context.Context
		cancel context.CancelFunc
	)

	ctxWt, cancel = context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWt, "/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "space", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseSpacesInfo(stdout)
}

// List the scheduling groups on the instance
func (c *Client) ListGroup(ctx context.Context, username string) ([]*GroupInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}

	var (
		ctxWt  context.Context
		cancel context.CancelFunc
	)

	ctxWt, cancel = context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWt, "/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "group", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseGroupsInfo(stdout)
}

// List the filesystems on the instance
func (c *Client) ListFS(ctx context.Context, username string) ([]*FSInfo, error) {
	unixUser, err := getUnixUser(username)
	if err != nil {
		return nil, err
	}

	var (
		ctxWt  context.Context
		cancel context.CancelFunc
	)

	ctxWt, cancel = context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWt, "/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "fs", "ls", "-m")
	stdout, _, err := c.execute(cmd)
	if err != nil {
		return nil, err
	}
	return c.parseFSsInfo(stdout)
}

func (c *Client) getEosMGMVersion(ctx context.Context) (string, error) {
	out, _, err := c.execute(exec.CommandContext(ctx, "/usr/bin/eos", "version"))
	if err != nil {
		return "", err
	}
	stdo_mgm := strings.Split(out, "\n")
	for _, l := range stdo_mgm {
		if strings.HasPrefix(l, "EOS_SERVER_VERSION=") {
			s := strings.Split(l, " ")
			return strings.Split(s[0], "EOS_SERVER_VERSION=")[1], nil
		}
	}
	return "", errors.New("version not found")
}

// List the version of different nodes in the instance
func (c *Client) ListVS(ctx context.Context) ([]*VSInfo, error) {

	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	mgmVersion, err := c.getEosMGMVersion(ctx)
	if err != nil {
		return nil, err
	}

	//cmd = exec.CommandContext(ctxWt, "/usr/bin/eos", "-r", unixUser.Uid, unixUser.Gid, "-b", "node", "ls","-m", "--sys", "|", "grep", "cern.ch", "|", "sort", "-t:", "-uk1,1")
	stdout, _, err := c.execute(exec.CommandContext(ctx, "/usr/bin/eos", "--json", "node", "ls"))
	if err != nil {
		return nil, err
	}

	nodeLSResponse := &NodeLSResponse{}
	err = json.Unmarshal([]byte(stdout), nodeLSResponse)
	if err != nil {
		return nil, err
	}

	return c.parseVSsInfo(mgmVersion, nodeLSResponse)
}

// List the activity of different users in the instance
func (c *Client) ListNS(ctx context.Context) ([]*NSInfo, []*NSActivityInfo, error) {

	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	stdout, _, err := c.execute(exec.CommandContext(ctx, "/usr/bin/eos", "ns", "stat", "-a", "-m"))
	if err != nil {
		return nil, nil, err
	}

	return c.parseNSsInfo(stdout)
}

func getHostname(hostport string) (string, string) {
	split := strings.Split(hostport, ":")
	return split[0], split[1]
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
		Hostport:              kv["hostport"],
		Status:                kv["status"],
		Nofs:                  kv["nofs"],
		SumStatStatfsFree:     kv["sum.stat.statfs.freebytes"],
		SumStatStatfsUsed:     kv["sum.stat.statfs.usedbytes"],
		SumStatStatfsTotal:    kv["sum.stat.statfs.capacity"],
		SumStatStatFilesFree:  kv["sum.stat.statfs.ffree"],
		SumStatStatFilesUsed:  kv["sum.stat.usedfiles"],
		SumStatStatFilesTotal: kv["sum.stat.statfs.files"],
		SumStatRopen:          kv["sum.stat.ropen"],
		SumStatWopen:          kv["sum.stat.wopen"],
		CfgStatSysThreads:     kv["cfg.stat.sys.threads"],
		SumStatNetInratemib:   kv["sum.stat.net.inratemib"],
		SumStatNetOutratemib:  kv["sum.stat.net.outratemib"],
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

// Gathers information of versions of nodes
func (c *Client) parseVSsInfo(mgmVersion string, nodeLSResponse *NodeLSResponse) ([]*VSInfo, error) {
	vsinfos := []*VSInfo{}

	if nodeLSResponse.ErrorMsg != "" {
		return nil, errors.New(nodeLSResponse.ErrorMsg)
	}

	//set := make(map[string]struct{})
	for _, node := range nodeLSResponse.Result {
		hostname, port := getHostname(node.HostPort)
		/* // Make sure each FST is only registered once.
		if _, ok := set[hostname]; ok {
			continue
		}
		set[hostname] = struct{}{}
		*/
		// Parse uptime to days
		s := strings.Split(node.Cfg.Stat.Sys.Uptime, "%20days,")[0]
		upt := strings.Split(s, "up%20")
		var uptime string
		if len(upt) < 2 {
			fmt.Println("Wrong uptime: ", upt)
			uptime = "0"
		} else {
			uptime = upt[1]
		}

		info := &VSInfo{
			EOSmgm:    mgmVersion,
			Hostname:  hostname,
			Port:      port,
			Geotag:    node.Cfg.Stat.Geotag,
			Vsize:     strconv.Itoa(node.Cfg.Stat.Sys.Vsize),
			Rss:       strconv.Itoa(node.Cfg.Stat.Sys.Rss),
			Threads:   strconv.Itoa(node.Cfg.Stat.Sys.Threads),
			Sockets:   strconv.Itoa(node.Cfg.Stat.Sys.Sockets),
			EOSfst:    node.Cfg.Stat.Sys.Eos.Version,
			Xrootdfst: node.Cfg.Stat.Sys.Xrootd.Version,
			KernelV:   node.Cfg.Stat.Sys.Kernel,
			Start:     node.Cfg.Stat.Sys.Eos.Start,
			Uptime:    uptime,
		}
		vsinfos = append(vsinfos, info)
	}

	return vsinfos, nil
}

// Gathers information of the namespace
func (c *Client) parseNSsInfo(raw string) ([]*NSInfo, []*NSActivityInfo, error) {
	var kv map[string]string
	var nsinfo *NSInfo
	var nsactinfo *NSActivityInfo
	nsinfos := []*NSInfo{}
	nsactinfos := []*NSActivityInfo{}
	rawLines := strings.Split(raw, "\n")
	for _, rl := range rawLines {
		if rl == "" {
			continue
		}
		kv = getMap(rl)
		// Only expose global data, without breakdown of users
		if kv["uid"] == "all" && kv["gid"] == "all" {
			// Separate activity info from namespace statistics info
			if _, ok := kv["cmd"]; ok {
				if kv["5s"] == "0.00" && kv["60s"] == "0.00" && kv["300s"] == "0.00" && kv["3600s"] == "0.00" {
				} else {
					nsactinfo = &NSActivityInfo{
						kv["uid"],
						kv["gid"],
						kv["cmd"],
						kv["total"],
						kv["5s"],
						kv["60s"],
						kv["300s"],
						kv["3600s"],
						kv["exec"],
						kv["execsig"],
						kv["exec99"],
						kv["execmax"],
					}
				}
			} else {
				if len(kv) <= 3 {
					for k := range kv {
						if k != "uid" && k != "gid" {
							if _, err := strconv.ParseFloat(kv[k], 64); err != nil {
								fmt.Sprintf("Value of '%s': '%s' is not floatable", k, kv[k])
							}
							nsinfo = &NSInfo{
								kv["ns.boot.file.time"],
								kv["ns.boot.status"],
								kv["ns.boot.time"],
								kv["ns.cache.containers.maxsize"],
								kv["ns.cache.containers.occupancy"],
								kv["ns.cache.files.maxsize"],
								kv["ns.cache.files.occupancy"],
								kv["ns.fds.all"],
								kv["ns.fusex.activeclients"],
								kv["ns.fusex.caps"],
								kv["ns.fusex.clients"],
								kv["ns.fusex.lockedclients"],
								kv["ns.latency.dirs"],
								kv["ns.latency.files"],
								kv["ns.latency.pending.updates"],
								kv["ns.latencypeak.eosviewmutex.1min"],
								kv["ns.latencypeak.eosviewmutex.2min"],
								kv["ns.latencypeak.eosviewmutex.5min"],
								kv["ns.latencypeak.eosviewmutex.last"],
								kv["ns.memory.growth"],
								kv["ns.memory.resident"],
								kv["ns.memory.share"],
								kv["ns.memory.virtual"],
								kv["ns.stat.threads"],
								kv["ns.total.directories"],
								kv["ns.total.directories.changelog.avg_entry_size"],
								kv["ns.total.directories.changelog.size"],
								kv["ns.total.files"],
								kv["ns.total.files.changelog.avg_entry_size"],
								kv["ns.total.files.changelog.size"],
								kv["ns.uptime"],
							}
						}
					}
				}
			}

		}
		if nsinfo != nil {
			nsinfos = append(nsinfos, nsinfo)
		}
		if nsactinfo != nil {
			nsactinfos = append(nsactinfos, nsactinfo)
		}
	}
	return nsinfos, nsactinfos, nil
}

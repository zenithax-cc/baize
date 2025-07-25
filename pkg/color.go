package pkg

var colorMap = map[string]map[string]map[string]string{
	"CPU": {
		"HyperThreading": {
			"Supported Disabled": colorRed,
			"Supported Enabled":  colorGreen,
			"Not Supported":      colorGreen,
		},
		"PowerState": {
			"Performance": colorGreen,
			"PowerSave":   colorRed,
		},
		"Diagnose": {
			"Healthy":   colorGreen,
			"Unhealthy": colorRed,
		},
		"DiagnoseDetail": {
			"default": colorRed,
		},
	},
	"Memory": {
		"Diagnose": {
			"Healthy":   colorGreen,
			"Unhealthy": colorRed,
		},
		"DiagnoseDetail": {
			"default": colorRed,
		},
	},
	"Bond": {
		"Status": {
			"up":   colorGreen,
			"down": colorRed,
		},
		"Diagnose": {
			"Healthy":   colorGreen,
			"Unhealthy": colorRed,
		},
		"DiagnoseDetail": {
			"default": colorRed,
		},
	},
	"Health": {
		"GameInit": {
			"ok":    colorGreen,
			"error": colorRed,
		},
		"Gpostd": {
			"ok":    colorGreen,
			"error": colorRed,
		},
		"Puppet": {
			"ok":    colorGreen,
			"error": colorRed,
		},
		"SSHPort": {
			"ok":    colorGreen,
			"error": colorRed,
		},
		"State": {
			"ok":    colorGreen,
			"error": colorRed,
		},
		"Errors": {
			"default": colorRed,
		},
		"ErrDetail": {
			"default": colorRed,
		},
	},
	"RAID": {
		"SystemDiskRAID": {
			"RAID1":   colorGreen,
			"default": colorRed,
		},
		"Diagnose": {
			"Healthy":   colorGreen,
			"Unhealthy": colorRed,
		},
		"DiagnoseDetail": {
			"default": colorRed,
		},
	},
}

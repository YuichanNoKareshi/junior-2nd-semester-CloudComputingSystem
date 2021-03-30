# qos lab
### Task one: implement a meter and a dropper
&emsp;&emsp;qos_meter_init和qos_meter_run主要参考DPDK/examples/qos_meter/main.c
```c++
int qos_meter_init(void)
{
    uint32_t i;
	int ret;

	ret = rte_meter_srtcm_profile_config(&app_srtcm_profile,
		&app_srtcm_params);
	if (ret)
		return ret;

	ret = rte_meter_trtcm_profile_config(&app_trtcm_profile,
		&app_trtcm_params);
	if (ret)
		return ret;

	for (i = 0; i < APP_FLOWS_MAX; i++) {
		ret = rte_meter_srtcm_config(&app_flows[i], &app_srtcm_profile);
		if (ret)
			return ret;
	}

	return 0;
}
```
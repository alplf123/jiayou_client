package common

type EventType string

const (
	EventEngineStart              EventType = "event_engine_start"
	EventEngineStop               EventType = "event_engine_stop"
	EventEngineJsStart            EventType = "event_engine_js_start"
	EventEngineJsEnd              EventType = "event_engine_js_end"
	EventEngineJsPanic            EventType = "event_engine_js_panic"
	EventProxyPoolError           EventType = "event_proxy_pool_error"
	EventProxyPoolExpired         EventType = "event_proxy_pool_expired"
	EventProxyPoolGrab            EventType = "event_proxy_pool_grab"
	EventProxyPoolVerify          EventType = "event_proxy_pool_verify"
	EventProxyPoolReady           EventType = "event_proxy_pool_ready"
	EventProxyPoolDone            EventType = "event_proxy_pool_done"
	EventDbReady                  EventType = "event_db_ready"
	EventDbDone                   EventType = "event_db_done"
	EventDbKeyExpired             EventType = "event_db_key_expired"
	EventEngineJsProxyGetter      EventType = "event_engine_js_proxy_getter"
	EventEngineJsProxySetter      EventType = "event_engine_js_proxy_setter"
	EventEngineJsProxyFunc        EventType = "event_engine_js_proxy_func"
	EventEngineJsProxyPanic       EventType = "event_engine_js_proxy_panic"
	EventCronDistributeBeforeTask EventType = "event_cron_distribute_before_task"
	EventCronDistributeAfterTask  EventType = "event_cron_distribute_after_task"
	EventCronDistributeHandleTask EventType = "event_cron_distribute_handle_task"
	EventSimulatorReady           EventType = "event_simulator_ready"
	EventSimulatorDone            EventType = "event_simulator_done"
	EventBrowserReady             EventType = "event_browser_ready"
	EventBrowserDone              EventType = "event_browser_done"
	EventOcrReady                 EventType = "event_ocr_ready"
	EventOcrDone                  EventType = "event_ocr_done"
	EventLoggerReady              EventType = "event_logger_ready"
	EventLocalCronReady           EventType = "event_local_cron_ready"
	EventLocalCronDone            EventType = "event_local_cron_done"
	EventDistributeCronReady      EventType = "event_distribute_cron_ready"
	EventDistributeCronDone       EventType = "event_distribute_cron_done"
	EventStorageReady             EventType = "event_storage_ready"
	EventStorageDone              EventType = "event_storage_done"
	EventFridaReady               EventType = "event_frida_ready"
	EventFridaDone                EventType = "event_frida_done"
	EventGinServerUp              EventType = "event_gin_server_up"
	EventGinServerReady           EventType = "event_gin_server_ready"
	EventReady                    EventType = "event_ready"
)

var EventSlices = []EventType{
	EventEngineStart,
	EventEngineStop,
	EventEngineJsStart,
	EventEngineJsEnd,
	EventEngineJsPanic,
	EventProxyPoolError,
	EventProxyPoolExpired,
	EventProxyPoolGrab,
	EventProxyPoolVerify,
	EventProxyPoolReady,
	EventProxyPoolDone,
	EventDbReady,
	EventDbDone,
	EventDbKeyExpired,
	EventEngineJsProxyGetter,
	EventEngineJsProxySetter,
	EventEngineJsProxyFunc,
	EventEngineJsProxyPanic,
	EventCronDistributeBeforeTask,
	EventCronDistributeAfterTask,
	EventCronDistributeHandleTask,
	EventSimulatorReady,
	EventSimulatorDone,
	EventBrowserReady,
	EventBrowserDone,
	EventOcrReady,
	EventOcrDone,
	EventLoggerReady,
	EventLocalCronReady,
	EventLocalCronDone,
	EventDistributeCronReady,
	EventDistributeCronDone,
	EventStorageReady,
	EventStorageDone,
	EventFridaReady,
	EventFridaDone,
	EventGinServerUp,
}

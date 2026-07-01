using TypeSpec.Http;

namespace OneMail;

/** Selectable window for the analytics overview */
enum SiteAnalyticsRange {
  `7d`,
  `30d`,
  `90d`,
}

/** Current contact counts (point-in-time snapshot, not range-scoped) */
model SiteAnalyticsContacts {
  /** All contacts in the workspace */
  total: int32;

  /** Contacts with status = active */
  active: int32;

  /** Contacts with status = unsubscribed */
  unsubscribed: int32;

  /** Contacts created within the selected range */
  newInRange: int32;
}

/**
 * Email engagement for the messages sent in the selected range (the send
 * cohort), from delivery logs. Opens/clicks are counted among that cohort, so
 * the counts reconcile and the rates stay in [0,1].
 */
model SiteAnalyticsEmail {
  /** Messages sent within the range */
  sentCount: int32;

  /** Messages sent in the range that were opened */
  openedCount: int32;

  /** Messages sent in the range that were clicked */
  clickedCount: int32;

  /** Opened ÷ sent (0–1) */
  openRate: float32;

  /** Clicked ÷ sent (0–1) */
  clickRate: float32;

  /** Clicked ÷ opened (0–1) */
  clickToOpenRate: float32;
}

/** Automation counts (point-in-time snapshot) */
model SiteAnalyticsAutomations {
  /** All automations in the workspace */
  total: int32;

  /** Automations with status = active */
  active: int32;

  /** Runs currently in progress */
  runsActive: int32;

  /** Runs that have completed */
  runsCompleted: int32;
}

/**
 * One UTC send-day of the cohort, zero-filled across the range. Opened/clicked
 * count the messages sent that day that were later opened/clicked.
 */
model SiteAnalyticsPoint {
  /** Day bucket, YYYY-MM-DD (UTC) */
  date: string;

  /** Messages sent that day */
  sent: int32;

  /** Messages sent that day that were opened */
  opened: int32;

  /** Messages sent that day that were clicked */
  clicked: int32;
}

/** Workspace analytics overview powering the dashboard */
model SiteAnalyticsOverview {
  contacts: SiteAnalyticsContacts;
  email: SiteAnalyticsEmail;
  automations: SiteAnalyticsAutomations;

  /** Daily engagement series, oldest first, one point per day in the range */
  timeseries: SiteAnalyticsPoint[];
}

// Not a CRUD resource — a single workspace-scoped aggregate with no key, so it
// keeps an explicit nested route rather than @parentResource/@autoRoute.
@route("/workspaces/{slug}/analytics")
@tag("Site Analytics")
namespace SiteAnalytics {
  /** Aggregate metrics for the workspace dashboard */
  @get
  @route("/overview")
  op overview(
    @path slug: string,

    /** Window to aggregate over (defaults to 30d) */
    @query range?: SiteAnalyticsRange,
  ): SiteAnalyticsOverview | NotFoundProblem;
}

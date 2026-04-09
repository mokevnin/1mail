using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact status */
enum ContactStatus {
  active,
  unsubscribed,
}

/** Contact resource */
@resource("contacts")
@segment("contacts")
@example(#{
  id: 101,
  email: "alice@example.com",
  firstName: "Alice",
  lastName: "Johnson",
  status: ContactStatus.active,
  timeZone: "America/New_York",
  customFields: #{
    plan: "pro",
    source: "import",
  },
  createdAt: "2026-04-01T10:15:30Z",
  updatedAt: "2026-04-03T08:45:00Z",
})
model ContactResource {
  /** Unique identifier */
  @key
  id: int64;

  /** Email address */
  email: EmailAddress;

  /** First name */
  firstName?: string;

  /** Last name */
  lastName?: string;

  /** Current status */
  status: ContactStatus;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName;

  /** Custom fields as key-value pairs */
  customFields?: Record<string>;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a contact */
@example(#{
  email: "alice@example.com",
  firstName: "Alice",
  lastName: "Johnson",
  timeZone: "America/New_York",
  customFields: #{
    plan: "pro",
    source: "import",
  },
})
model CreateContactInput {
  /** Email address */
  email: EmailAddress;

  /** First name */
  firstName?: string;

  /** Last name */
  lastName?: string;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName;

  /** Custom fields as key-value pairs */
  customFields?: Record<string>;
}

/** Request body for updating a contact */
@example(#{
  firstName: "Alicia",
  timeZone: "Europe/Berlin",
  customFields: #{
    plan: "enterprise",
  },
})
model UpdateContactInput {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

@example(#{
  content: #[
    #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alice",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "America/New_York",
      customFields: #{
        plan: "pro",
        source: "import",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-03T08:45:00Z",
    },
  ],
  pageable: #{
    pageSize: 25,
    pageNumber: 0,
    offset: 0,
    paged: true,
    unpaged: false,
    sort: #{
      sorted: true,
      unsorted: false,
      empty: false,
    },
  },
  totalElements: 1,
  totalPages: 1,
  size: 25,
  number: 0,
  numberOfElements: 1,
  first: true,
  last: true,
  empty: false,
  sort: #{
    sorted: true,
    unsorted: false,
    empty: false,
  },
})
model ContactPage is Page<ContactResource> {}

@route("/contacts")
@tag("Contacts")
namespace Contacts {
  /** Create a contact */
  @createsResource(ContactResource)
  @post
  @opExample(#{
    parameters: #{
      body: #{
        email: "alice@example.com",
        firstName: "Alice",
        lastName: "Johnson",
        timeZone: "America/New_York",
        customFields: #{
          plan: "pro",
          source: "import",
        },
      },
    },
    returnType: #{
      statusCode: 201,
      body: #{
        id: 101,
        email: "alice@example.com",
        firstName: "Alice",
        lastName: "Johnson",
        status: ContactStatus.active,
        timeZone: "America/New_York",
        customFields: #{
          plan: "pro",
          source: "import",
        },
        createdAt: "2026-04-01T10:15:30Z",
        updatedAt: "2026-04-03T08:45:00Z",
      },
    },
  }, #{ title: "Create contact" })
  op create(@body body: CreateContactInput): {
    @statusCode statusCode: 201;
    @body body: ContactResource;
  } | UnauthorizedProblem | ConflictProblem | UnprocessableEntityProblem;

  /** List contacts with pagination */
  @listsResource(ContactResource)
  @get
  @opExample(#{
    parameters: #{
      page: 0,
      size: 25,
      sort: #["email,asc"],
      status: ContactStatus.active,
    },
    returnType: #{
      content: #[
        #{
          id: 101,
          email: "alice@example.com",
          firstName: "Alice",
          lastName: "Johnson",
          status: ContactStatus.active,
          timeZone: "America/New_York",
          customFields: #{
            plan: "pro",
            source: "import",
          },
          createdAt: "2026-04-01T10:15:30Z",
          updatedAt: "2026-04-03T08:45:00Z",
        },
      ],
      pageable: #{
        pageSize: 25,
        pageNumber: 0,
        offset: 0,
        paged: true,
        unpaged: false,
        sort: #{
          sorted: true,
          unsorted: false,
          empty: false,
        },
      },
      totalElements: 1,
      totalPages: 1,
      size: 25,
      number: 0,
      numberOfElements: 1,
      first: true,
      last: true,
      empty: false,
      sort: #{
        sorted: true,
        unsorted: false,
        empty: false,
      },
    },
  }, #{ title: "List contacts" })
  op list(
    ...PageQuery,

    /** Filter by status */
    @query status?: ContactStatus,
  ): ContactPage | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a contact by ID */
  @readsResource(ContactResource)
  @get
  @autoRoute
  @opExample(#{
    parameters: #{
      id: 101,
    },
    returnType: #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alice",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "America/New_York",
      customFields: #{
        plan: "pro",
        source: "import",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-03T08:45:00Z",
    },
  }, #{ title: "Get contact" })
  op get(@path id: int64): ContactResource | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;

  /** Update a contact */
  @updatesResource(ContactResource)
  @put
  @autoRoute
  @opExample(#{
    parameters: #{
      id: 101,
      body: #{
        firstName: "Alicia",
        timeZone: "Europe/Berlin",
        customFields: #{
          plan: "enterprise",
        },
      },
    },
    returnType: #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alicia",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "Europe/Berlin",
      customFields: #{
        plan: "enterprise",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-06T12:00:00Z",
    },
  }, #{ title: "Update contact" })
  op update(@path id: int64, @body body: UpdateContactInput): ContactResource | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | ConflictProblem | UnprocessableEntityProblem;

  /** Delete a contact */
  @deletesResource(ContactResource)
  @delete
  @autoRoute
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

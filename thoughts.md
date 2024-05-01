- timestamps are stored in postgres tables as the "timestamp with time zone" data type, and when they are
  fetched from the db with Go code, the resulting time.Time values also take into account that stored timezone.
  time.Now() returns the current time.Time value with the current local timezone in my mind. so it's safe to comapare "timestamp with time zone" postgres values with time.Now() values from Go code. (comparison methods properly take into account associated timezone values of the two time.Time values)
  - *maybe* it would be better to store time values in the db without the timezone, but always in UTC.
    i.e. with the "timestamp [without time zone]" postgres data type. when storing time.Time values in 
    the db that way, the time.Time value would first have to be converted to the UTC timezone, i.e.
    (time.Time).UTC(). then when fetching the time value from the database it will be interpreted as 
    an UTC value (which it is) and we could safely compared it with any other time.Time values,
    including a time.Now(), which is returned in the local timezone, but the time package can safely
    compare Time values of different timezones
  - for now I opted to use the first approach, i.e. store timestamps with the timezone in postgres

- "Continue with Spotify" login flow explained:
  1. User is logged out, opens app, clicks "Continue with Spotify" button
  2. Frontend app makes request to /api/account/spotify-auth-url?flow_type=continue_with
    1. The "flow_type" query parameter in this request is required to be one of "continue_with" or "connect" - and depending on its value the redirect
    URL back from Spotify changes.
    2. A random "state" string is generated, saved as a Cookie on the client, and also appended as an URL query parameter to the Spotify redirect auth. URL
  3. The server responds with a Spotify auth. URL to which the user is then redirected to in the browser. The user authenticates with Spotify, and if everything is correct, is redirected from it back to the musicdash website, with a "#spotify_connect_with" hash present at the end of the URL. Three URL query parameters are present: state, code, and potentially error. The "state" parameter is the same one server generated in step 2. The "code" parameter is used for further authentication.
  4. The frontend app makes a new POST request to /api/account/spotify_continue_with, forwarding to it the 3 query parameters from the previous step. Once the server receives this request, it first valides the state in the query paramater (i.e. the one from Spotify) against the one saved as a cookie on the client. This is a security measure against tampering. 
  5. The server follows the next steps of the "Authorization Code" auth. flow: it uses the "code" parameter from Spotify alongside the app's Client ID and Client Secret to obtain an Authentication Token (and refresh token, etc...) that can then be used to interact with the API.
  6. The server polls the Spotify API ant obtains details about the current Spotify user's profile. It then checks if there's an existing musicdash account linked with the Spotify profile. If there is, the user is logged into it. If there isn't, a new boilerplate musicdash account is created using the info from the current Spotify profile, and the two accounts are linked. The user is then logged into the brand new musicdash account.
  All further uses of "Continue with Spotify"... will then lead to the client simply being instantly logged into this new account.
  
  That's the gist of how the 1-click "Continue with Spotify" flow works. The procedure for linking a Spotify profile with a an existing musicdash account is very similar so I won't document it. Basically in step 2. "flow_type" is set to "connect" with leads to the frontend hash after the redirect being "spotify_connect" after which the client makes a request to /api/account/spotify_link_account, forwarding the parameters from the priorspotify request, just like in the Continue With flow. All in all pretty simple.
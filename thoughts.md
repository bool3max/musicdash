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
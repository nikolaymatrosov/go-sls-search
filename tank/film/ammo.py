# -*- coding: utf-8 -*---
import itertools
import random
import sys
import urllib.parse

years = range(2000, 2018)
countries = [
    "сша",
    "франция",
    "германия",
    "великобритания",
    "канада",
    "италия",
]
durations = [random.randint(60, 120) for i in range(10)]
age_limits = [0, 12, 16, 18]


def make_ammo(method, url, headers):
    """ makes phantom ammo """
    # http request w/o entity body template
    req_template = (
        "%s %s HTTP/1.1\r\n"
        "%s\r\n"
        "\r\n"
    )

    req = req_template % (method, url, headers)

    # phantom ammo template
    ammo_template = (
        "%d\n"
        "%s"
    )

    return ammo_template % (len(req), req)


def main():
    for year, country, duration, age_limit in itertools.product(years, countries, durations, age_limits):
        method = "GET"

        qs = '+countryOfProduction:{country} +crYearOfProduction={year} +duration:>={duration} +ageLimit:{age_limit}'.format(
            country=country,
            year=year,
            duration=duration,
            age_limit=age_limit,
        )
        query = urllib.parse.quote(qs, safe='')
        url = "/d4e53gmg6vb62fq3f0ri?term=" + query

        headers = "Host: functions.yandexcloud.net\r\n" + \
                  "User-Agent: tank\r\n" + \
                  "Accept: */*\r\n" + \
                  "Connection: Close"

        sys.stdout.write(make_ammo(method, url, headers))


if __name__ == "__main__":
    main()
# Spring Auto-reconfiguration Framework

⚠️ **DEPRECATED**: This framework is disabled by default as of December 2025. Please migrate to [java-cfenv](framework-java_cf_env.md).

The Spring Auto-reconfiguration Framework causes an application to be automatically reconfigured to work with configured cloud services.

## Deprecation Notice

**Spring Auto-reconfiguration has been deprecated** since July 2019 when Spring Cloud Connectors entered maintenance mode. This framework is now **disabled by default**.

**Migration Path**:
- For **Spring Boot 3.x** applications: Use [java-cfenv](https://github.com/pivotal-cf/java-cfenv) (see [java-cfenv framework docs](framework-java_cf_env.md))
- For **Spring Boot 2.x** applications: Migrate to java-cfenv when upgrading to Spring Boot 3.x

**To re-enable** (not recommended):
```bash
cf set-env my-app JBP_CONFIG_SPRING_AUTO_RECONFIGURATION '{enabled: true}'
```

<table>
  <tr>
    <td><strong>Detection Criterion</strong></td>
    <td>Existence of a <tt>spring-core*.jar</tt> file in the application directory AND explicitly enabled via configuration</td>
  </tr>
  <tr>
    <td><strong>Tags</strong></td>
    <td><tt>spring-auto-reconfiguration=&lt;version&gt;</tt> (only when enabled)</td>
  </tr>
  <tr>
    <td><strong>Default</strong></td>
    <td><strong>DISABLED</strong> (as of Dec 2025)</td>
  </tr>
</table>
Tags are printed to standard output by the buildpack detect script

The Spring Auto-reconfiguration Framework adds the `cloud` profile to any existing Spring profiles such as those defined in the [`SPRING_PROFILES_ACTIVE`][] environment variable.  It also uses the [Spring Cloud Cloud Foundry Connector][] to replace any bean of a candidate type with one mapped to a bound service instance.  Please see the [Auto-Reconfiguration][] project for more details.

## Configuration
For general information on configuring the buildpack, including how to specify configuration values through environment variables, refer to [Configuration and Extension][].

The framework can be configured by modifying the [`config/spring_auto_reconfiguration.yml`][] file in the buildpack fork.  The framework uses the [`Repository` utility support][repositories] and so it supports the [version syntax][] defined there.

| Name | Description
| ---- | -----------
| `enabled` | Whether to attempt auto-reconfiguration. **Default: `false`** (disabled since Dec 2025)
| `repository_root` | The URL of the Auto-reconfiguration repository index ([details][repositories]).
| `version` | The version of Auto-reconfiguration to use. Candidate versions can be found in [this listing][].

### Enabling Spring Auto-reconfiguration

To enable this deprecated framework, set the environment variable:

```bash
cf set-env my-app JBP_CONFIG_SPRING_AUTO_RECONFIGURATION '{enabled: true}'
cf restage my-app
```

**Warning**: You will see deprecation warnings in your logs when this framework is enabled.

[Auto-Reconfiguration]: https://github.com/cloudfoundry/java-buildpack-auto-reconfiguration
[Configuration and Extension]: ../README.md#configuration-and-extension
[`config/spring_auto_reconfiguration.yml`]: ../config/spring_auto_reconfiguration.yml
[repositories]: extending-repositories.md
[Spring Cloud Cloud Foundry Connector]: https://cloud.spring.io/spring-cloud-connectors/spring-cloud-cloud-foundry-connector.html
[this listing]: http://download.pivotal.io.s3.amazonaws.com/auto-reconfiguration/index.yml
[version syntax]: extending-repositories.md#version-syntax-and-ordering
[`SPRING_PROFILES_ACTIVE`]: http://docs.spring.io/spring/docs/4.0.0.RELEASE/javadoc-api/org/springframework/core/env/AbstractEnvironment.html#ACTIVE_PROFILES_PROPERTY_NAME

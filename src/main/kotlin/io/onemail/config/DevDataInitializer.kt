package io.onemail.config

import io.onemail.account.Account
import io.onemail.account.AccountRepository
import io.onemail.token.ApiToken
import io.onemail.token.ApiTokenRepository
import org.springframework.boot.ApplicationArguments
import org.springframework.boot.ApplicationRunner
import org.springframework.context.annotation.Profile
import org.springframework.stereotype.Component

@Profile("dev")
@Component
class DevDataInitializer(
    private val accountRepository: AccountRepository,
    private val apiTokenRepository: ApiTokenRepository,
) : ApplicationRunner {
    override fun run(args: ApplicationArguments) {
        val account = Account(name = "Development")
        val savedAccount = accountRepository.save(account)
        val apiToken =
            ApiToken(
                account = savedAccount,
                token = "onemail-dev-token",
                name = "Development Token",
            )
        apiTokenRepository.save(apiToken)
    }
}

<#import "template.ftl" as layout>
<link href='https://fonts.googleapis.com/css?family=Figtree' rel='stylesheet'>
<div class="layout-simple">
    <div id="main" class="grid12">
        <div class="grid7">
            <div class="width-limited">
                <div class="stacklok-logo"></div>
                <div class="title">Sign Up to<br>Unlock Additional Possibilities</div>
                <div class="subtitle">Start with Stacklok Account!</div>
            </div>
            <div class="width-limited">
            <@layout.registrationLayout; section>
                <#if section = "title">
                    ${msg("registerWithTitle",(realm.displayName!''))}
                <#elseif section = "header">
                    ${msg("registerWithTitle",(realm.displayName!''))}
                <#elseif section = "form">
                    <form id="kc-register-form" class="${properties.kcFormClass!}" action="${url.registrationAction}" method="post">
                    <input type="text" readonly value="this is not a login form" style="display: none;">
                    <input type="password" readonly value="this is not a login form" style="display: none;">

                    <#if !realm.registrationEmailAsUsername>
                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('username',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="username" class="${properties.kcLabelClass!} label">${msg("username")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="text" id="username" class="${properties.kcInputClass!} input" name="username" value="${(register.formData.username!'')}" />
                            </div>
                        </div>
                    </#if>
                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('firstName',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="firstName" class="${properties.kcLabelClass!} label">${msg("firstName")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="text" id="firstName" class="${properties.kcInputClass!} input" name="firstName" value="${(register.formData.firstName!'')}" />
                            </div>
                        </div>

                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('lastName',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="lastName" class="${properties.kcLabelClass!} label">${msg("lastName")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="text" id="lastName" class="${properties.kcInputClass!} input" name="lastName" value="${(register.formData.lastName!'')}" />
                            </div>
                        </div>

                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('email',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="email" class="${properties.kcLabelClass!} label">${msg("email")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="text" id="email" class="${properties.kcInputClass!} input" name="email" value="${(register.formData.email!'')}" />
                            </div>
                        </div>

                        <#if passwordRequired>
                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('password',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="password" class="${properties.kcLabelClass!} label">${msg("password")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="password" id="password" class="${properties.kcInputClass!} input" name="password" />
                            </div>
                        </div>

                        <div class="${properties.kcFormGroupClass!} ${messagesPerField.printIfExists('password-confirm',properties.kcFormGroupErrorClass!)}">
                            <div class="${properties.kcLabelWrapperClass!}">
                                <label for="password-confirm" class="${properties.kcLabelClass!} label">${msg("passwordConfirm")}</label>
                            </div>
                            <div class="${properties.kcInputWrapperClass!}">
                                <input type="password" id="password-confirm" class="${properties.kcInputClass!} input" name="password-confirm" />
                            </div>
                        </div>
                        </#if>

                        <#if recaptchaRequired??>
                        <div class="form-group">
                            <div class="${properties.kcInputWrapperClass!}">
                                <div class="g-recaptcha" data-size="compact" data-sitekey="${recaptchaSiteKey}"></div>
                            </div>
                        </div>
                        </#if>

                        <div class="${properties.kcFormGroupClass!}">
                            <div id="kc-form-options" class="${properties.kcFormOptionsClass!}">
                                <div class="${properties.kcFormOptionsWrapperClass!}">
                                    <span><a href="${url.loginUrl}" class="link">Back to Login Page</a></span>
                                </div>
                            </div>

                            <div id="kc-form-buttons" class="${properties.kcFormButtonsClass!}">
                                <input class="button" type="submit" value="${msg("doRegister")}"/>
                            </div>
                        </div>
                    </form>
                </#if>
            </@layout.registrationLayout>
            </div>
        </div>

        <div class="grid5">
            <div class="img"></div>
        </div>
    </div>
</div>
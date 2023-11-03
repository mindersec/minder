# Frequently Asked Questions

## Can you tell me more about Stacklok, the company behind Minder?

Stacklok’s mission is to make it easier for developers to build more trustworthy software. Our free-to-use products, [Trusty](https://bit.ly/stackloktrusty) and [Minder](https://bit.ly/stacklokminder), help developers make safer dependency choices and help development teams and open source maintainers adopt safer development practices. 

Our co-founders, Craig McLuckie and Luke Hinds, are veterans of the open source and software security communities. Craig McLuckie co-founded [Kubernetes](http://kubernetes.io), an open source system for automating deployment, scaling, and management of containerized applications, and Luke Hinds founded [Sigstore](http://sigstore.dev), an open source project that dramatically simplifies how developers sign and verify software artifacts. 

Learn more about us at [www.stacklok.com](http://www.stacklok.com). 

## Do I have to use Minder to improve my Trusty Score?

No, you don’t. Minder can make it easier to put in place security policies and practices that may improve your Trusty Score, but use of Minder doesn’t guarantee a better Trusty Score, and we don’t give favorable treatment or higher scores to project owners just for using it. Project owners can take measures outside of Minder—like increasing their commit frequency, the number of contributors to their project, or the number of contributions they make to other projects—that can improve their Trusty Score.    

## How do Trusty and Minder work together? Are they integrated?

Yes—Trusty and Minder are complementary tools. For example, in Minder, you can set a policy to block pull requests that contain dependencies with low Trusty scores. When this happens, Minder will also display a list of alternative packages with their Trusty scores, to help developers select a safer option. 

## What value does Minder add above GitHub Advanced Security?

Minder integrates with GitHub security features such as Dependabot and Code Scanning to make it easy to manage many repositories with a consistent set of policies. Minder’s Projects concept (on the roadmap) will allow you to group multiple repositories and apply policy consistently. In addition, Minder’s policy engine enables autoremediation of any configuration gaps, so you can automatically fix configuration issues across repositories.

## Why is the Minder mascot a marmot? 

Marmots look out for each other: when one marmot leaves its burrow to eat, another marmot will go with it to act as a lookout. If it sees a threat, it will whistle to alert other marmots in the area about possible danger. We want Minder to be your trusted sidekick, looking out for risk and keeping your software safe. 
